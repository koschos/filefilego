package storage

import (
	"bufio"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"io"
	"math/big"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"time"

	"github.com/filefilego/filefilego/common"
	"github.com/filefilego/filefilego/common/hexutil"
	"github.com/filefilego/filefilego/crypto"
	"github.com/filefilego/filefilego/database"
	"github.com/filefilego/filefilego/node/protocols/messages"
	"github.com/libp2p/go-libp2p/core/network"
	log "github.com/sirupsen/logrus"
	"github.com/syndtr/goleveldb/leveldb/util"
	"google.golang.org/protobuf/proto"
)

const (

	// AdminAccess represents full access.
	AdminAccess = "admin"
	// UserAccess represents privilege access.
	UserAccess = "user"

	tokenAccessHours      = 2160
	tokenPrefix           = "token"
	fileHashPrefix        = "mdt"
	fileHashSortingPrefix = "mds"
	fileHashCountPrefix   = "fileHashCount"

	bufferSize = 8192
)

// Interface defines the functionalities of the storage engine.
type Interface interface {
	StoragePath() string
	Enabled() bool
	SetEnabled(val bool)
	CreateSubfolders() (string, error)
	SaveToken(token AccessToken) error
	SaveFileMetadata(fileHash, peerID string, metadata FileMetadata) error
	DeleteFileMetadata(fileHash, peerID string) error
	GetFileMetadata(fileHash string, peerID string) (FileMetadata, error)
	CanAccess(token string) (bool, AccessToken, error)
	HandleIncomingFileUploads(stream network.Stream)
	ListFiles(currentPage, pageSize int, order string) ([]FileMetadataWithDBKey, uint64, error)
	DeleteFileFromDB(key string) error
	ExportFiles() ([]FileMetadataWithDBKey, error)
	ImportFiles(string) (int, error)
}

// CreateStorageAccessTokenRequest represents the request for creating a new storage token.
type CreateStorageAccessTokenRequest struct {
	UserAccessType string `json:"user_access_type"`
}

// FileMetadata holds the metadata for a file.
type FileMetadata struct {
	FileName       string `json:"file_name"`
	MerkleRootHash string `json:"merkle_root_hash"`
	Hash           string `json:"hash"`
	FilePath       string `json:"file_path"`
	Size           int64  `json:"size"`
	RemotePeer     string `json:"remote_peer"`
	Timestamp      int64  `json:"timestamp"`
	FeesPerByte    string `json:"fees_per_byte"`
	PublicKeyOwner string `json:"public_key_owner"`
}

// FileMetadataWithDBKey holds the file metatada and the key.
type FileMetadataWithDBKey struct {
	FileMetadata
	Key              string `json:"key"`
	FileExistsOnNode bool   `json:"file_exists_on_node"`
}

// AccessToken represents an access token.
type AccessToken struct {
	AccessType string `json:"access_type"`
	Token      string `json:"token"`
	ExpiresAt  int64  `json:"expires_at"`
}

// IntrospectAccessTokenResponse contains the token info and some info about the storage node.
type IntrospectAccessTokenResponse struct {
	AccessToken       AccessToken `json:"access_token"`
	AllowFeesOverride bool        `json:"allow_fees_override"`
	Platform          string      `json:"platform"`
	FeesPerByte       string      `json:"fees_per_byte"`
	PublicKey         string      `json:"public_key"`
	StorageCapacity   uint64      `json:"storage_capacity"`
	PeerID            string      `json:"storage_provider_peer_addr"`
	UptimeSeconds     int64       `json:"uptime_seconds"`
}

// Storage represents the storage engine and the metadata.
type Storage struct {
	db                      database.Database
	storagePath             string
	enabled                 bool
	merkleTreeTotalSegments int
	peerID                  string
	allowFeesOverride       bool
	publicKey               string
	feesPerByte             string
	uptime                  int64
}

// New creates a new storage instance.
func New(db database.Database, storagePath string, enabled bool, adminToken string, merkleTreeTotalSegments int, peerID string, allowFeesOverride bool, publicKey, feesPerByte string, uptime int64) (*Storage, error) {
	if db == nil {
		return nil, errors.New("db is nil")
	}

	if storagePath == "" {
		return nil, errors.New("storagePath is empty")
	}

	if adminToken == "" {
		return nil, errors.New("adminToken is empty")
	}

	if merkleTreeTotalSegments == 0 {
		return nil, errors.New("merkle tree total segments is zero")
	}

	if !common.DirExists(storagePath) {
		if err := common.CreateDirectory(storagePath); err != nil {
			return nil, fmt.Errorf("failed to create storage directory: %w", err)
		}
	}

	if peerID == "" {
		return nil, errors.New("peerID is empty")
	}

	storage := &Storage{
		db:                      db,
		storagePath:             storagePath,
		enabled:                 enabled,
		merkleTreeTotalSegments: merkleTreeTotalSegments,
		peerID:                  peerID,
		allowFeesOverride:       allowFeesOverride,
		publicKey:               publicKey,
		feesPerByte:             feesPerByte,
		uptime:                  uptime,
	}

	token := AccessToken{
		AccessType: AdminAccess,
		Token:      adminToken,
		ExpiresAt:  time.Now().Add(time.Hour * tokenAccessHours).Unix(),
	}

	if err := storage.SaveToken(token); err != nil {
		return nil, fmt.Errorf("failed to save admin token: %w", err)
	}

	return storage, nil
}

// StoragePath return the storage path.
func (s *Storage) StoragePath() string {
	return s.storagePath
}

// Enabled return if storage functionality is enabled.
func (s *Storage) Enabled() bool {
	return s.enabled
}

// SetEnabled return if storage functionality is enabled.
func (s *Storage) SetEnabled(val bool) {
	s.enabled = val
}

// CreateSubfolders creates sub folders with current date inside the data directory.
func (s *Storage) CreateSubfolders() (string, error) {
	currentTime := time.Now()
	folder := fmt.Sprintf("%d-%02d-%02d", currentTime.Year(), currentTime.Month(), currentTime.Day())
	destinationPath := filepath.Join(s.storagePath, folder)
	if err := common.CreateDirectory(destinationPath); err != nil {
		return "", err
	}
	return destinationPath, nil
}

// SaveToken saves an access token into the database.
func (s *Storage) SaveToken(token AccessToken) error {
	if token.Token == "" || token.AccessType == "" || token.ExpiresAt == 0 {
		return errors.New("invalid access token")
	}
	data, err := json.Marshal(&token)
	if err != nil {
		return fmt.Errorf("failed to marshal access token: %w", err)
	}

	return s.db.Put(append([]byte(tokenPrefix), []byte(token.Token)...), data)
}

// GetTotalFilesStored returns the total number of files stored on this node.
func (s *Storage) GetTotalFilesStored() uint64 {
	countBytes, err := s.db.Get([]byte(fileHashCountPrefix))
	if err != nil || len(countBytes) != 8 {
		return 0
	}

	return binary.BigEndian.Uint64(countBytes)
}

// DeleteFileMetadata deletes the  metadata from the db.
func (s *Storage) DeleteFileMetadata(fileHash, peerID string) error {
	prefix := append([]byte(fileHashPrefix), []byte(fileHash)...)
	err := s.db.Delete(append(prefix, []byte(peerID)...))
	if err != nil {
		return fmt.Errorf("failed to delete file metadata: %w", err)
	}
	return nil
}

// SaveFileMetadata saves a file's metadata in the database.
func (s *Storage) SaveFileMetadata(fileHash, peerID string, metadata FileMetadata) error {
	if metadata.MerkleRootHash == "" || metadata.Hash == "" || metadata.Size == 0 {
		return errors.New("invalid file metadata")
	}

	data, err := json.Marshal(&metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal file metadata: %w", err)
	}

	// save using: filehash + peerID
	prefix := append([]byte(fileHashPrefix), []byte(fileHash)...)
	err = s.db.Put(append(prefix, []byte(peerID)...), data)
	if err != nil {
		return fmt.Errorf("failed to insert filehash %s: %w", fileHash, err)
	}

	idx := s.GetTotalFilesStored()
	idx++
	itemsUint64 := make([]byte, 8)
	binary.BigEndian.PutUint64(itemsUint64, idx)

	err = s.db.Put([]byte(fileHashCountPrefix), itemsUint64)
	if err != nil {
		return fmt.Errorf("failed to save files count: %w", err)
	}

	// save using: count + filehash + peerid
	prefixSorting := append([]byte(fileHashSortingPrefix), itemsUint64...)
	prefixSorting = append(prefixSorting, []byte(fileHash)...)
	err = s.db.Put(append(prefixSorting, []byte(peerID)...), []byte{})
	if err != nil {
		return fmt.Errorf("failed to insert filehash to sorted table %s: %w", fileHash, err)
	}

	return nil
}

// GetFileMetadata gets a file's metadata in the database.
func (s *Storage) GetFileMetadata(fileHash string, peerID string) (FileMetadata, error) {
	if fileHash == "" {
		return FileMetadata{}, errors.New("file hash is empty")
	}

	prefix := append([]byte(fileHashPrefix), []byte(fileHash)...)
	data, err := s.db.Get(append(prefix, []byte(peerID)...))
	if err != nil {
		return FileMetadata{}, fmt.Errorf("failed to get file metadata: %w", err)
	}
	metadata := FileMetadata{}
	err = json.Unmarshal(data, &metadata)
	if err != nil {
		return FileMetadata{}, fmt.Errorf("failed to unmarshal file metadata: %w", err)
	}
	return metadata, nil
}

// DeleteFileFromDB deletes a file from local db.
func (s *Storage) DeleteFileFromDB(key string) error {
	keyBytes, err := hexutil.Decode(key)
	if err != nil {
		return fmt.Errorf("failed to decode key: %w", err)
	}

	// check if the file is locally available
	hash := string(keyBytes[8+len([]byte(fileHashSortingPrefix)):])
	item, err := s.GetFileMetadata(hash[:40], hash[40:])
	if err != nil {
		return fmt.Errorf("failed to find file metadata: %w", err)
	}
	if common.FileExists(item.FilePath) {
		err = os.Remove(item.FilePath)
		if err != nil {
			return fmt.Errorf("failed to remove file from local node: %w", err)
		}
	}

	err = s.db.Delete(keyBytes)
	if err != nil {
		return fmt.Errorf("failed to delete from db: %w", err)
	}

	idx := s.GetTotalFilesStored()
	if idx == 0 {
		idx = 0
	} else {
		idx--
	}

	itemsUint64 := make([]byte, 8)
	binary.BigEndian.PutUint64(itemsUint64, idx)

	err = s.db.Put([]byte(fileHashCountPrefix), itemsUint64)
	if err != nil {
		return fmt.Errorf("failed to decrement saved files count: %w", err)
	}

	fileHash := hash[:40]
	peerID := hash[40:]

	err = s.DeleteFileMetadata(fileHash, peerID)
	if err != nil {
		return fmt.Errorf("failed to delete file from db: %w", err)
	}

	return nil
}

// ExportFiles gets all the uploaded files.
func (s *Storage) ExportFiles() ([]FileMetadataWithDBKey, error) {
	iter := s.db.NewIterator(util.BytesPrefix([]byte(fileHashSortingPrefix)), nil)
	items := make([]FileMetadataWithDBKey, 0)
	for iter.Next() {
		key := iter.Key()
		if len(key) == 0 {
			break
		}

		hash := string(key[8+len([]byte(fileHashSortingPrefix)):])
		item, err := s.GetFileMetadata(hash[:40], hash[40:])
		if err != nil {
			continue
		}

		kitem := FileMetadataWithDBKey{
			FileMetadata: item,
			Key:          hexutil.Encode(key),
		}

		if common.FileExists(item.FilePath) {
			kitem.FileExistsOnNode = true
		}

		items = append(items, kitem)
	}

	iter.Release()
	err := iter.Error()
	if err != nil {
		return nil, fmt.Errorf("failed to release uploaded files iterator: %w", err)
	}

	return items, nil
}

// ImportFiles gets all the uploaded files list
func (s *Storage) ImportFiles(importedFile string) (int, error) {
	data, err := os.ReadFile(importedFile)
	if err != nil {
		return 0, fmt.Errorf("failed to read import file content: %w", err)
	}
	files := make([]FileMetadataWithDBKey, 0)
	err = json.Unmarshal(data, &files)
	if err != nil {
		return 0, fmt.Errorf("failed to unmarshal file metadata: %w", err)
	}

	imported := 0
	for _, v := range files {
		k, err := hexutil.Decode(v.Key)
		if err != nil {
			log.Warnf("failed to marshal decode key: %v", err)
			continue
		}

		hash := string(k[8+len([]byte(fileHashSortingPrefix)):])
		fileHash := hash[:40]
		peerID := hash[40:]

		m, err := s.GetFileMetadata(fileHash, peerID)
		if err != nil {
			err = s.SaveFileMetadata(fileHash, peerID, v.FileMetadata)
			if err != nil {
				log.Warnf("failed to restore file info: %v", err)
			} else {
				imported++
			}
		} else {
			log.Warnf("item already exists %v", m)
		}
	}

	return imported, nil
}

// ListFiles the uploaded files.
func (s *Storage) ListFiles(currentPage, pageSize int, order string) ([]FileMetadataWithDBKey, uint64, error) {
	if currentPage < 0 {
		currentPage = 0
	}

	if pageSize == 0 {
		pageSize = 10
	} else if pageSize > 1000 {
		pageSize = 1000
	}

	start := (currentPage) * pageSize
	if start < 0 {
		start = 0
	}

	limit := pageSize
	iter := s.db.NewIterator(util.BytesPrefix([]byte(fileHashSortingPrefix)), nil)
	items := make([]FileMetadataWithDBKey, 0)
	i := 0

	if order == "asc" {
		for iter.Next() {
			i++
			if limit == 0 {
				break
			}

			if i < start {
				continue
			}

			key := iter.Key()
			if len(key) == 0 {
				break
			}

			hash := string(key[8+len([]byte(fileHashSortingPrefix)):])
			item, err := s.GetFileMetadata(hash[:40], hash[40:])
			if err != nil {
				continue
			}

			kitem := FileMetadataWithDBKey{
				FileMetadata: item,
				Key:          hexutil.Encode(key),
			}

			if common.FileExists(item.FilePath) {
				kitem.FileExistsOnNode = true
			}

			items = append(items, kitem)
			limit--
		}
	} else {
		if iter.Last() {
			key := iter.Key()
			i++
			if len(key) != 0 && i > start {
				hash := string(key[8+len([]byte(fileHashSortingPrefix)):])
				item, err := s.GetFileMetadata(hash[:40], hash[40:])
				if err == nil {
					kitem := FileMetadataWithDBKey{
						FileMetadata: item,
						Key:          hexutil.Encode(key),
					}

					if common.FileExists(item.FilePath) {
						kitem.FileExistsOnNode = true
					}

					items = append(items, kitem)
					limit--
				}
			}
		}

		for iter.Prev() {
			i++
			if limit == 0 {
				break
			}

			if i < start {
				continue
			}

			key := iter.Key()
			if len(key) == 0 {
				break
			}

			hash := string(key[8+len([]byte(fileHashSortingPrefix)):])
			item, err := s.GetFileMetadata(hash[:40], hash[40:])
			if err != nil {
				continue
			}

			kitem := FileMetadataWithDBKey{
				FileMetadata: item,
				Key:          hexutil.Encode(key),
			}

			if common.FileExists(item.FilePath) {
				kitem.FileExistsOnNode = true
			}

			items = append(items, kitem)
			limit--
		}
	}

	iter.Release()
	err := iter.Error()
	if err != nil {
		return nil, 0, fmt.Errorf("failed to release uploaded files iterator: %w", err)
	}

	idx := s.GetTotalFilesStored()

	return items, idx, nil
}

// CanAccess authorizes access.
func (s *Storage) CanAccess(token string) (bool, AccessToken, error) {
	if token == "" {
		return false, AccessToken{}, errors.New("token is empty")
	}

	data, err := s.db.Get(append([]byte(tokenPrefix), []byte(token)...))
	if err != nil {
		return false, AccessToken{}, err
	}

	accToken := AccessToken{}
	err = json.Unmarshal(data, &accToken)
	if err != nil {
		return false, AccessToken{}, fmt.Errorf("failed to unmarshal access token: %w", err)
	}

	if time.Now().Unix() > accToken.ExpiresAt {
		return false, AccessToken{}, errors.New("access token is expired")
	}

	return true, accToken, nil
}

func writeHeaderPayload(w http.ResponseWriter, status int, payload string) {
	w.WriteHeader(status)
	// nolint:errcheck
	w.Write([]byte(payload))
}

// ServeHTTP handles file uploading.
func (s *Storage) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
	w.Header().Set("Access-Control-Allow-Headers", "Accept, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization")

	if r.Method == http.MethodOptions {
		return
	}

	if !s.enabled {
		writeHeaderPayload(w, http.StatusForbidden, `{"error": "storage is not enabled"}`)
		return
	}

	if r.Method != http.MethodPost {
		writeHeaderPayload(w, http.StatusMethodNotAllowed, `{"error": "method not available"}`)
		return
	}

	can, userAccessToken, err := s.CanAccess(r.Header.Get("Authorization"))
	if !can {
		writeHeaderPayload(w, http.StatusForbidden, `{"error": "`+err.Error()+`"}`)
		return
	}

	reader, err := r.MultipartReader()
	if err != nil {
		writeHeaderPayload(w, http.StatusInternalServerError, `{"error": "`+err.Error()+`"}`)
		return
	}

	folderPath, err := s.CreateSubfolders()
	if err != nil {
		writeHeaderPayload(w, http.StatusInternalServerError, `{"error": "`+err.Error()+`"}`)
		return
	}

	tmpFileHex := ""
	fileName := ""
	publicKeyOwner := ""
	fileFeesPerByte := ""
	for {
		part, err := reader.NextPart()
		if err == io.EOF {
			// done reading body
			break
		}

		formName := part.FormName()

		if formName == "public_key_owner" {
			pubKey, err := io.ReadAll(part)
			if err != nil {
				log.Warnf("failed to read owner's public key from multipart: %v", err)
			}
			publicKeyOwner = hexutil.Encode(pubKey)
			continue
		}

		if formName == "fees_per_byte" {
			feesPerByte, err := io.ReadAll(part)
			if err != nil {
				log.Warnf("failed to read fees per byte for multipart: %v", err)
			}

			fileFeesPerByte = string(feesPerByte)

			_, ok := big.NewInt(0).SetString(fileFeesPerByte, 10)
			if !ok {
				fileFeesPerByte = ""
			}

			continue
		}

		if formName == "file" {
			fileName = part.FileName()
			tmpFileName, err := crypto.RandomEntropy(40)
			if err != nil {
				writeHeaderPayload(w, http.StatusInternalServerError, `{"error": "`+err.Error()+`"}`)
				return
			}

			tmpFileHex = hexutil.Encode(tmpFileName)
			destFile, err := os.Create(filepath.Join(folderPath, tmpFileHex))
			if err != nil {
				writeHeaderPayload(w, http.StatusInternalServerError, `{"error": "failed to open file on the system"}`)
				return
			}
			_, err = io.Copy(destFile, part)
			if err != nil {
				writeHeaderPayload(w, http.StatusInternalServerError, `{"error": "failed to copy from multipart reader"}`)
				return
			}
			destFile.Close()
		}
		part.Close()
	}

	if !validateFileName(fileName) {
		if err != nil {
			writeHeaderPayload(w, http.StatusInternalServerError, `{"error": "file name is invalid"}`)
			return
		}
	}

	old := filepath.Join(folderPath, tmpFileHex)
	fileSize, err := common.FileSize(old)
	if err != nil {
		os.Remove(old)
		writeHeaderPayload(w, http.StatusInternalServerError, `{"error": "failed to get file's size"}`)
		return
	}

	if fileSize <= 0 {
		os.Remove(old)
		writeHeaderPayload(w, http.StatusInternalServerError, `{"error": "file size is zero"}`)
		return
	}

	fHash, err := crypto.Sha1File(old)
	if err != nil {
		os.Remove(old)
		writeHeaderPayload(w, http.StatusInternalServerError, `{"error": "failed to hash contents of file"}`)
		return
	}

	fileMetadata, err := s.GetFileMetadata(fHash, s.peerID)
	if err == nil {
		os.Remove(old)
		writeHeaderPayload(w, http.StatusOK, fmt.Sprintf(`{"file_name":"%s","file_hash": "%s", "merkle_root_hash": "%s", "size": %d}`, fileMetadata.FileName, fileMetadata.Hash, fileMetadata.MerkleRootHash, fileMetadata.Size))
		return
	}

	newPath := filepath.Join(folderPath, fHash)
	err = os.Rename(old, newPath)
	if err != nil {
		log.Errorf("failed to move uploaded file: %v", err)
		os.Remove(old)
		writeHeaderPayload(w, http.StatusInternalServerError, `{"error": "failed to move uploaded file"}`)
		return
	}

	howManySegments, _, _, _, err := common.FileSegmentsInfo(int(fileSize), s.merkleTreeTotalSegments, 0)
	if err != nil {
		log.Errorf("failed to get file segment info: %v", err)
		os.Remove(old)
		writeHeaderPayload(w, http.StatusInternalServerError, `{"error": "failed to get file segment info"}`)
		return
	}

	orderedSlice := make([]int, howManySegments)
	for i := 0; i < howManySegments; i++ {
		orderedSlice[i] = i
	}

	fMerkleRootHash, err := common.GetFileMerkleRootHash(newPath, s.merkleTreeTotalSegments, orderedSlice)
	if err != nil {
		os.Remove(old)
		writeHeaderPayload(w, http.StatusInternalServerError, `{"error": "failed to get merkle root hash"}`)
		return
	}

	// if access token is of UserAccess type AND this node accepts dynamic fethen remove any
	// fees associated with this upload
	if userAccessToken.AccessType == UserAccess && !s.allowFeesOverride {
		fileFeesPerByte = ""
	}

	fileName = html.EscapeString(fileName)
	fileMetadata = FileMetadata{
		FileName:       fileName,
		MerkleRootHash: hexutil.Encode(fMerkleRootHash),
		Hash:           fHash,
		FilePath:       newPath,
		Size:           fileSize,
		Timestamp:      time.Now().Unix(),
		PublicKeyOwner: publicKeyOwner,
		FeesPerByte:    fileFeesPerByte,
	}

	err = s.SaveFileMetadata(fHash, s.peerID, fileMetadata)
	if err != nil {
		os.Remove(newPath)
		writeHeaderPayload(w, http.StatusInternalServerError, `{"error": "`+err.Error()+`"}`)
		return
	}

	writeHeaderPayload(w, http.StatusOK, fmt.Sprintf(`{"file_name":"%s","file_hash": "%s", "merkle_root_hash": "%s", "size": %d}`, fileMetadata.FileName, fileMetadata.Hash, fileMetadata.MerkleRootHash, fileMetadata.Size))
}

// ServeHTTP handles file uploading.
func (s *Storage) HandleIncomingFileUploads(stream network.Stream) {
	defer func() {
		if r := recover(); r != nil {
			log.Errorf("recovering panic from HandleIncomingFileUploads %v", r)
		}
	}()

	c := bufio.NewReader(stream)
	defer stream.Close()

	// read the first 8 bytes to determine the size of the message
	msgLengthBuffer := make([]byte, 8)
	_, err := c.Read(msgLengthBuffer)
	if err != nil {
		log.Errorf("failed to read from HandleIncomingFileUploads stream: %v", err)
		return
	}

	// create a buffer with the size of the message and then read until its full
	lengthPrefix := int64(binary.LittleEndian.Uint64(msgLengthBuffer))
	buf := make([]byte, lengthPrefix)

	// read the full message
	_, err = io.ReadFull(c, buf)
	if err != nil {
		log.Errorf("failed to read from HandleIncomingFileUploads stream to buffer: %v", err)
		return
	}

	request := messages.StorageFileUploadMetadataProto{}
	if err := proto.Unmarshal(buf, &request); err != nil {
		log.Errorf("failed to unmarshall data from HandleIncomingFileUploads stream: %v", err)
		return
	}

	// if invalid fees set it to empty
	_, ok := big.NewInt(0).SetString(request.FeesPerByte, 10)
	if !ok {
		request.FeesPerByte = ""
	}

	fileName := request.FileName

	if !validateFileName(fileName) {
		if err != nil {
			return
		}
	}

	folderPath, err := s.CreateSubfolders()
	if err != nil {
		log.Errorf("failed to create subfolders: %v", err)
		return
	}

	tmpFileName, err := crypto.RandomEntropy(40)
	if err != nil {
		log.Errorf("failed to create a random temp file: %v", err)
		return
	}

	tmpFileHex := hexutil.Encode(tmpFileName)
	destFile, err := os.Create(filepath.Join(folderPath, tmpFileHex))
	if err != nil {
		log.Errorf("failed to create destination file: %v", err)
		return
	}

	// handle upload
	buf = make([]byte, bufferSize)
	for {
		n, err := stream.Read(buf)
		if n > 0 {
			wroteN, err := destFile.Write(buf[:n])
			if wroteN != n || err != nil {
				log.Errorf("failed to write to destination file (buf: %d, output: %d): %v", n, wroteN, err)
				destFile.Close()
				return
			}
		}

		if err == io.EOF {
			break
		}

		if err != nil {
			log.Errorf("fialed to read file content to buffer: %v", err)
			break
		}
	}
	destFile.Close()

	old := filepath.Join(folderPath, tmpFileHex)
	fileSize, err := common.FileSize(old)
	if err != nil {
		log.Errorf("failed to get destination file size: %v", err)
		os.Remove(old)
		return
	}

	if fileSize <= 0 {
		log.Error("file size is zero")
		os.Remove(old)
		return
	}

	fHash, err := crypto.Sha1File(old)
	if err != nil {
		log.Errorf("failed to hash destination file: %v", err)
		os.Remove(old)
		return
	}

	fileMetadata, err := s.GetFileMetadata(fHash, s.peerID)
	if err == nil {
		os.Remove(old)
		fileMetadataBytes, err := json.Marshal(fileMetadata)
		if err != nil {
			log.Errorf("failed to marshal file metadata: %v", err)
		}

		_, err = stream.Write(fileMetadataBytes)
		if err != nil {
			log.Errorf("failed to write the uploaded file metadata bytes to the stream: %v", err)
		}

		return
	}

	newPath := filepath.Join(folderPath, fHash)

	if !common.FileExists(newPath) {
		err = os.Rename(old, newPath)
		if err != nil {
			log.Errorf("failed to move uploaded file: %v", err)
			os.Remove(old)
			return
		}
	} else {
		os.Remove(old)
	}

	howManySegments, _, _, _, err := common.FileSegmentsInfo(int(fileSize), s.merkleTreeTotalSegments, 0)
	if err != nil {
		log.Errorf("failed to get file segment info: segment: %d filesize: %d : %v", howManySegments, fileSize, err)
		return
	}

	if howManySegments <= 0 {
		log.Errorf("invalid file segment: %d filesize: %d", howManySegments, fileSize)
		os.Remove(newPath)
		return
	}

	orderedSlice := make([]int, howManySegments)
	for i := 0; i < howManySegments; i++ {
		orderedSlice[i] = i
	}

	fMerkleRootHash, err := common.GetFileMerkleRootHash(newPath, s.merkleTreeTotalSegments, orderedSlice)
	if err != nil {
		log.Errorf("failed to get file merkle root hash: %v", err)
		os.Remove(old)
		return
	}

	feesPerByte := ""
	if request.FeesPerByte != "" && s.allowFeesOverride {
		feesPerByte = request.FeesPerByte
	}

	fileName = html.EscapeString(fileName)
	fileMetadata = FileMetadata{
		FileName:       fileName,
		MerkleRootHash: hexutil.Encode(fMerkleRootHash),
		Hash:           fHash,
		FilePath:       newPath,
		Size:           fileSize,
		Timestamp:      time.Now().Unix(),
		PublicKeyOwner: hexutil.Encode(request.PublicKeyOwner),
		FeesPerByte:    feesPerByte,
	}

	err = s.SaveFileMetadata(fHash, s.peerID, fileMetadata)
	if err != nil {
		log.Errorf("failed to save file metadata: %v", err)
		os.Remove(newPath)
		return
	}

	// send back the result to uploader
	// remove the local file path
	fileMetadata.FilePath = ""
	fileMetadataBytes, err := json.Marshal(fileMetadata)
	if err != nil {
		log.Errorf("failed to marshal file metadata: %v", err)
		os.Remove(newPath)
		return
	}

	_, err = stream.Write(fileMetadataBytes)
	if err != nil {
		log.Errorf("failed to write the uploaded file metadata bytes to the stream: %v", err)
		os.Remove(newPath)
		return
	}
}

// CreateStorageAccessToken creates a storage access token.
func (s *Storage) CreateStorageAccessToken(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
	w.Header().Set("Access-Control-Allow-Headers", "Accept, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization")

	if r.Method == http.MethodOptions {
		return
	}

	if !s.enabled {
		writeHeaderPayload(w, http.StatusForbidden, `{"error": "storage is not enabled"}`)
		return
	}

	if r.Method != http.MethodPost {
		writeHeaderPayload(w, http.StatusMethodNotAllowed, `{"error": "method not available"}`)
		return
	}

	can, accessToken, err := s.CanAccess(r.Header.Get("Authorization"))
	if !can {
		writeHeaderPayload(w, http.StatusForbidden, `{"error": "`+err.Error()+`"}`)
		return
	}

	if accessToken.AccessType != AdminAccess {
		writeHeaderPayload(w, http.StatusUnauthorized, `{"error": "not authorized to perform this operation"}`)
		return
	}

	randomBytes, err := crypto.RandomEntropy(60)
	if err != nil {
		writeHeaderPayload(w, http.StatusInternalServerError, `{"error": "`+err.Error()+`"}`)
		return
	}

	AccessType := UserAccess

	userType := CreateStorageAccessTokenRequest{}
	maxBytes := int64(5 * common.KB)
	limitReader := io.LimitReader(r.Body, maxBytes)
	buffer, err := io.ReadAll(limitReader)
	if err != nil {
		writeHeaderPayload(w, http.StatusInternalServerError, `{"error": "`+err.Error()+`"}`)
		return
	}
	r.Body.Close()

	// ignore error
	_ = json.Unmarshal(buffer, &userType)

	if userType.UserAccessType == "admin" {
		AccessType = AdminAccess
	}

	// create a user token
	token := AccessToken{
		AccessType: AccessType,
		Token:      hexutil.Encode(randomBytes),
		ExpiresAt:  time.Now().Add(time.Hour * tokenAccessHours).Unix(),
	}

	err = s.SaveToken(token)
	if err != nil {
		writeHeaderPayload(w, http.StatusInternalServerError, `{"error": "`+err.Error()+`"}`)
		return
	}

	writeHeaderPayload(w, http.StatusOK, `{"token": "`+token.Token+`"}`)
}

// IntrospectAccessToken returns the payload of an access token.
func (s *Storage) IntrospectAccessToken(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
	w.Header().Set("Access-Control-Allow-Headers", "Accept, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization")

	if r.Method == http.MethodOptions {
		return
	}

	if !s.enabled {
		writeHeaderPayload(w, http.StatusForbidden, `{"error": "storage is not enabled"}`)
		return
	}

	if r.Method != http.MethodPost {
		writeHeaderPayload(w, http.StatusMethodNotAllowed, `{"error": "method not available"}`)
		return
	}

	can, accessToken, err := s.CanAccess(r.Header.Get("Authorization"))
	if !can {
		writeHeaderPayload(w, http.StatusForbidden, `{"error": "`+err.Error()+`"}`)
		return
	}

	payload := IntrospectAccessTokenResponse{
		AccessToken:       accessToken,
		AllowFeesOverride: s.allowFeesOverride,
		Platform:          runtime.GOOS,
		UptimeSeconds:     time.Now().Unix() - s.uptime,
		FeesPerByte:       s.feesPerByte,
		PublicKey:         s.publicKey,
		PeerID:            s.peerID,
	}

	storageDirCapacity, err := common.GetDirectoryFreeSpace(s.storagePath)
	if err == nil {
		payload.StorageCapacity = storageDirCapacity
	}

	data, err := json.Marshal(payload)
	if err != nil {
		writeHeaderPayload(w, http.StatusForbidden, `{"error": "`+err.Error()+`"}`)
		return
	}

	writeHeaderPayload(w, http.StatusOK, string(data))
}

func validateFileName(fileName string) bool {
	// Clean the file name
	fileName = filepath.Clean(fileName)

	// Define a regular expression pattern for valid file names
	pattern := "^[^<>:\"/\\\\|?*\\x00-\\x1F]+(?: [^<>:\"/\\\\|?*\\x00-\\x1F]+)*$"

	// Compile the regular expression
	regex, err := regexp.Compile(pattern)
	if err != nil {
		// Handle the error
		return false
	}

	// Check if the file name matches the pattern
	return regex.MatchString(fileName)
}
