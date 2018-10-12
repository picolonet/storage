package picolo

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"github.com/Pallinder/sillyname-go"
	ma "github.com/multiformats/go-multiaddr"
	log "github.com/sirupsen/logrus"
	"io/ioutil"
	"net"
	"net/http"
	"path/filepath"
)

//Elliptic curve to use
var curve = elliptic.P256()

const publicIpUrl = "https://api.ipify.org"

var PicNode *PicoloNode
var nodeInfo *NodeInfo

func InitNode() {
	log.Info("Initializing node")
	PicNode = new(PicoloNode)
	assignNameAndId()
	//get node's network info
	PicNode.NetInfo = initNet()
	//get current cpu load
	PicNode.Load = getCpuLoad()
	//get disk stats
	PicNode.TotalDisk, PicNode.FreeDisk = getDiskStats()
	//get memory stats
	PicNode.TotalMemory, PicNode.FreeMem = getMemStats()
}

func assignNameAndId() {
	if registered() {
		node, err := ioutil.ReadFile(filepath.Join(PicoloDataDir, NodeInfoFile))
		if err != nil {
			log.Fatalf("Reading node info file failed, %v", err)
		}
		err = json.Unmarshal(node, &nodeInfo)
		if err != nil {
			log.Fatalf("Unmarshalling node info file failed, %v", err)
		}
		PicNode.Id = nodeInfo.id
		PicNode.Name = nodeInfo.name
	} else {
		PicNode.Name = sillyname.GenerateStupidName()
		PicNode.Id = generateId(PicNode.Name)
	}
}

func getCpuLoad() (load uint8) {
	// todo implement
	return
}

func getDiskStats() (totalDisk int64, freeDisk int64) {
	// todo implement
	return
}

func getMemStats() (totalMem int64, freeMem int64) {
	// todo implement
	return
}

func initNet() *NetworkInfo {
	//get public ip addresses
	res, err := http.Get(publicIpUrl)
	if err != nil {
		log.Errorf("Error getting public ip: %v", err)
	}
	publicIp, err := ioutil.ReadAll(res.Body)
	if err != nil {
		log.Errorf("Error reading public ip response: %v", err)
	}
	log.Infof("Public ip: %s", publicIp)

	// Get preferred outbound ip of this machine
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		log.Error(err)
	}
	defer conn.Close()
	localAddr := conn.LocalAddr().(*net.UDPAddr)
	log.Infof("Private ip: %s", localAddr.IP)
	m1, _ := ma.NewMultiaddr("/ip4/" + string(publicIp) + "/tcp/0")
	m2, _ := ma.NewMultiaddr("/ip4/" + localAddr.IP.String() + "/tcp/0")
	m3, _ := ma.NewMultiaddr("/ip6/::1/tcp/0")
	log.Infof("Multi addresses: %s, %s, %s: ", m1, m2, m3)

	netMap := new(NetworkInfo)
	netMap.PublicIp4 = string(publicIp)
	netMap.PrivateIp4 = localAddr.IP.String()
	netMap.PublicIp6 = m3.String()

	return netMap

	//get network interfaces
	/*interfaces, err := net.Interfaces()
	if err != nil {
		log.Errorf("Error getting network interfaces: %v", err)
	}
	for _, i := range interfaces {
		interfaceName, err := net.InterfaceByName(i.Name)
		if err != nil {
			log.Errorf("Error getting network interface %s: %v", i.Name, err)
		}
		addresses, err := interfaceName.Addrs()
		if err != nil {
			log.Errorf("Error getting network addresses %v", err)
		}
		for _, addr := range addresses {
			log.Infof("Interface Address is %v", addr.String())
		}
	}*/

}

func generateId(nodeName string) string {
	pubkeyCurve := curve

	privKey := new(ecdsa.PrivateKey)
	privKey, err := ecdsa.GenerateKey(pubkeyCurve, rand.Reader) // this generates a public & private key pair

	if err != nil {
		log.Fatalf("Error generating private & public key pair: %v", err)
	}

	pubKey := privKey.PublicKey

	nodeId := pubKeyToAddress(pubKey)
	log.Infof("Node Id: %v", nodeId)

	saveNodeInfo(privKey, &pubKey, nodeId, nodeName)

	return nodeId
}

// saves keys to the given file with
// restrictive permissions. The key data is saved hex-encoded.
func saveNodeInfo(privKey *ecdsa.PrivateKey, pubKey *ecdsa.PublicKey, nodeId string, nodeName string) error {
	file := filepath.Join(PicoloDataDir, NodeInfoFile)
	privHex := hex.EncodeToString(marshalPrivkey(privKey))
	pubHex := hex.EncodeToString(marshalPubkey(pubKey))
	nodeInfo := &NodeInfo{
		id:         nodeId,
		name:       nodeName,
		privateKey: privHex,
		publicKey:  pubHex,
	}
	jsonData, err := json.Marshal(nodeInfo)
	if err != nil {
		log.Fatalf("Error marshaling node info %v", err)
	}
	return ioutil.WriteFile(file, jsonData, 0600)
}

func pubKeyToAddress(pubKey ecdsa.PublicKey) string {
	bytes := marshalPubkey(&pubKey)
	sha256 := sha256.New()
	sha256.Write([]byte(bytes))
	return hex.EncodeToString(sha256.Sum(nil))
}

// FromECDSA exports a private key into a binary dump.
func marshalPrivkey(priv *ecdsa.PrivateKey) []byte {
	if priv == nil {
		return nil
	}
	return priv.D.Bytes()
}

func marshalPubkey(pub *ecdsa.PublicKey) []byte {
	if pub == nil || pub.X == nil || pub.Y == nil {
		return nil
	}
	return elliptic.Marshal(curve, pub.X, pub.Y)
}

func unMarshalPubkey(pub []byte) (*ecdsa.PublicKey, error) {
	x, y := elliptic.Unmarshal(curve, pub)
	if x == nil {
		return nil, errors.New("invalid public key")
	}
	return &ecdsa.PublicKey{Curve: curve, X: x, Y: y}, nil
}