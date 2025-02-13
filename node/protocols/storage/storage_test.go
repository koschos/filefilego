package storage

import (
	"bytes"
	"context"
	"crypto/sha256"
	"fmt"
	"os"
	"runtime"
	"testing"
	"time"

	"github.com/filefilego/filefilego/common"
	ffgcrypto "github.com/filefilego/filefilego/crypto"
	"github.com/filefilego/filefilego/database"
	"github.com/filefilego/filefilego/node/protocols/messages"
	internalstorage "github.com/filefilego/filefilego/storage"
	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	connmgr "github.com/libp2p/go-libp2p/p2p/net/connmgr"
	noise "github.com/libp2p/go-libp2p/p2p/security/noise"
	libp2ptls "github.com/libp2p/go-libp2p/p2p/security/tls"
	"github.com/stretchr/testify/assert"
	"github.com/syndtr/goleveldb/leveldb"
)

func TestStorageProtocol(t *testing.T) {
	h1, _, pubKey := newHost(t, "7365")
	h2, _, _ := newHost(t, "7366")

	db, err := leveldb.OpenFile("storage.db", nil)
	assert.NoError(t, err)
	driver, err := database.New(db)
	assert.NoError(t, err)
	storagePath := "storagePath"
	t.Cleanup(func() {
		db.Close()
		os.RemoveAll("storage.db")
		os.RemoveAll(storagePath)
	})

	storage, err := internalstorage.New(driver, storagePath, true, "admintoken", 1024, h1.ID().String(), false, "pubkey", "10", time.Now().Unix())
	assert.NoError(t, err)
	uptime := time.Now().Unix()

	protocol1, err := New(nil, storage, nil, true, uptime, false, "0", false)
	assert.EqualError(t, err, "host is nil")
	assert.Nil(t, protocol1)

	protocol1, err = New(h1, storage, nil, true, uptime, false, "0", false)
	assert.NoError(t, err)
	assert.NotNil(t, protocol1)
	protocol2, err := New(h2, storage, nil, true, uptime, false, "0", false)
	assert.NoError(t, err)
	assert.NotNil(t, protocol2)

	peer2Info := peer.AddrInfo{
		ID:    h2.ID(),
		Addrs: h2.Addrs(),
	}

	err = h1.Connect(context.Background(), peer2Info)
	assert.NoError(t, err)

	timePassed, err := protocol1.TestSpeedWithRemotePeer(context.TODO(), h2.ID(), 10*common.MB)
	assert.NoError(t, err)
	ms1, err := time.ParseDuration("1ms")
	assert.NoError(t, err)
	assert.Greater(t, timePassed, ms1)

	throughput := calculateThroughput(10*common.MB, timePassed)
	assert.Greater(t, throughput, float64(1))

	pubKeyBytes, err := pubKey.Raw()
	assert.NoError(t, err)

	response := &messages.StorageQueryResponseProto{
		StorageProviderPeerAddr: h1.ID().Pretty(),
		Location:                "US",
		FeesPerByte:             "0x01",
		PublicKey:               pubKeyBytes,
		Uptime:                  30,
		StorageCapacity:         10,
		Platform:                runtime.GOOS,
	}

	data := bytes.Join(
		[][]byte{
			[]byte(response.StorageProviderPeerAddr),
			[]byte(response.Location),
			[]byte(response.FeesPerByte),
			response.PublicKey,
			[]byte(fmt.Sprintf("%d", response.StorageCapacity)),
			[]byte(fmt.Sprintf("%d", response.Uptime)),
		},
		[]byte{},
	)

	h := sha256.New()
	if _, err := h.Write(data); err != nil {
		assert.NoError(t, err)
	}
	hash := h.Sum(nil)
	privateKey := h1.Peerstore().PrivKey(h1.ID())
	sig, err := privateKey.Sign(hash)
	assert.NoError(t, err)

	response.Hash = make([]byte, len(hash))
	response.Signature = make([]byte, len(sig))

	copy(response.Hash, hash)
	copy(response.Signature, sig)

	err = protocol1.SendStorageQueryResponse(context.TODO(), h2.ID(), response)
	assert.NoError(t, err)
	err = protocol1.SendStorageQueryResponse(context.TODO(), h2.ID(), response)
	assert.NoError(t, err)

	time.Sleep(1 * time.Second)
	providers := protocol2.GetDiscoveredStorageProviders()
	assert.Len(t, providers, 1)

	numPeers, err := protocol1.SendDiscoveredStorageTransferRequest(context.TODO(), h2.ID())
	assert.NoError(t, err)
	assert.Equal(t, 1, numPeers)

	fhash, err := ffgcrypto.Sha1File("storage.go")
	assert.NoError(t, err)
	assert.NotEmpty(t, fhash)

	// uploads file with cancelled context
	cancelCtx, cancel := context.WithCancel(context.TODO())
	cancel()

	pubKeyFileOwner, err := pubKey.Raw()
	assert.NoError(t, err)

	_, err = protocol1.UploadFileWithMetadata(cancelCtx, h2.ID(), "storage.go", pubKeyFileOwner, "")
	assert.Error(t, err)

	// reset the upload progress so we can reupload
	protocol1.uploadProgress = make(map[string]int)

	fhashremote, err := protocol1.UploadFileWithMetadata(context.TODO(), h2.ID(), "storage.go", pubKeyFileOwner, "")
	assert.NoError(t, err)
	assert.Equal(t, fhash, fhashremote.Hash)
	time.Sleep(1 * time.Second)

	metadata, err := protocol2.storage.GetFileMetadata(fhash, protocol1.host.ID().String())
	assert.NoError(t, err)
	assert.Equal(t, "storage.go", metadata.FileName)
	assert.NotEmpty(t, metadata.FilePath)
	assert.NotEmpty(t, metadata.Hash)
	assert.NotEmpty(t, metadata.MerkleRootHash)
	assert.NotEmpty(t, metadata.Size)
	uploadedData, fhash, err := protocol1.GetUploadProgress(h2.ID(), "storage.go")
	assert.NoError(t, err)
	assert.NotEqual(t, 0, uploadedData)
	assert.NotEmpty(t, fhash)

	capabilities, err := protocol1.GetStorageCapabilities(context.TODO(), h2.ID())
	assert.NoError(t, err)
	assert.NotNil(t, capabilities)
	assert.Equal(t, "0", capabilities.FeesPerByte)
}

func newHost(t *testing.T, port string) (host.Host, crypto.PrivKey, crypto.PubKey) {
	priv, pubKey, err := crypto.GenerateKeyPair(crypto.Secp256k1, 256)
	assert.NoError(t, err)
	connManager, err := connmgr.NewConnManager(
		100,
		400,
		connmgr.WithGracePeriod(time.Minute),
	)
	assert.NoError(t, err)

	host, err := libp2p.New(libp2p.Identity(priv),
		libp2p.ListenAddrStrings(fmt.Sprintf("/ip4/127.0.0.1/tcp/%s", port)),
		libp2p.Ping(false),
		libp2p.Security(libp2ptls.ID, libp2ptls.New),
		libp2p.Security(noise.ID, noise.New),
		libp2p.DefaultTransports,
		libp2p.ConnectionManager(connManager),
		libp2p.NATPortMap(),
		libp2p.EnableNATService(),
	)
	assert.NoError(t, err)
	return host, priv, pubKey
}

func calculateThroughput(fileSize uint64, duration time.Duration) float64 {
	bytesPerSecond := float64(fileSize) / duration.Seconds()
	return bytesPerSecond / (1024 * 1024) // convert to MB/s
}
