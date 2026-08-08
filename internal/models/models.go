package models

import "time"

type ServerConfig struct {
	ID          string `yaml:"id" json:"id"`
	Name        string `yaml:"name" json:"name"`
	Host        string `yaml:"host" json:"host"`
	Port        int    `yaml:"port" json:"port"`
	Username    string `yaml:"username" json:"username"`
	AuthMethod  string `yaml:"authMethod" json:"authMethod"`
	Password    string `yaml:"password,omitempty" json:"password,omitempty"`
	PrivateKey  string `yaml:"privateKey,omitempty" json:"privateKey,omitempty"`
	Passphrase  string `yaml:"passphrase,omitempty" json:"passphrase,omitempty"`
	ViaServerID string `yaml:"viaServerId,omitempty" json:"viaServerId,omitempty"`
	IsLocal     bool   `yaml:"isLocal,omitempty" json:"isLocal,omitempty"`
}

type LocalConfig struct {
	Username   string `yaml:"username" json:"username"`
	Password   string `yaml:"password,omitempty" json:"password,omitempty"`
	Configured bool   `yaml:"-" json:"configured"`
}

type WGInterface struct {
	Name       string   `yaml:"name" json:"name"`
	PublicKey  string   `yaml:"publicKey" json:"publicKey"`
	PrivateKey string   `yaml:"privateKey,omitempty" json:"privateKey,omitempty"`
	ListenPort int      `yaml:"listenPort" json:"listenPort"`
	Endpoint   string   `yaml:"endpoint,omitempty" json:"endpoint,omitempty"`
	RxBytes    int64    `yaml:"rxBytes" json:"rxBytes"`
	TxBytes    int64    `yaml:"txBytes" json:"txBytes"`
	Peers      []WGPeer `yaml:"peers" json:"peers"`
	Online     bool     `yaml:"online" json:"online"`
}

type WGPeer struct {
	PublicKey           string    `yaml:"publicKey" json:"publicKey"`
	PresharedKey        string    `yaml:"presharedKey,omitempty" json:"presharedKey,omitempty"`
	Endpoint            string    `yaml:"endpoint,omitempty" json:"endpoint,omitempty"`
	AllowedIPs          []string  `yaml:"allowedIPs" json:"allowedIPs"`
	LatestHandshake     time.Time `yaml:"latestHandshake" json:"latestHandshake"`
	RxBytes             int64     `yaml:"rxBytes" json:"rxBytes"`
	TxBytes             int64     `yaml:"txBytes" json:"txBytes"`
	PersistentKeepalive int       `yaml:"persistentKeepalive,omitempty" json:"persistentKeepalive,omitempty"`
	Name                string    `yaml:"name,omitempty" json:"name,omitempty"`
	Description         string    `yaml:"description,omitempty" json:"description,omitempty"`
}

type WGStatus struct {
	Interfaces     []WGInterface `yaml:"interfaces" json:"interfaces"`
	Hostname       string        `yaml:"hostname,omitempty" json:"hostname,omitempty"`
	ServerIP       string        `yaml:"serverIP,omitempty" json:"serverIP,omitempty"`
	OS             string        `yaml:"os,omitempty" json:"os,omitempty"`
	PackageManager string        `yaml:"packageManager,omitempty" json:"packageManager,omitempty"`
	WGNotInstalled bool          `yaml:"wgNotInstalled,omitempty" json:"wgNotInstalled,omitempty"`
}

type AddPeerRequest struct {
	ServerID            string   `yaml:"serverId" json:"serverId"`
	Interface           string   `yaml:"interface" json:"interface"`
	PublicKey           string   `yaml:"publicKey" json:"publicKey"`
	AllowedIPs          []string `yaml:"allowedIPs" json:"allowedIPs"`
	PresharedKey        string   `yaml:"presharedKey,omitempty" json:"presharedKey,omitempty"`
	Endpoint            string   `yaml:"endpoint,omitempty" json:"endpoint,omitempty"`
	PersistentKeepalive int      `yaml:"persistentKeepalive,omitempty" json:"persistentKeepalive,omitempty"`
	Name                string   `yaml:"name,omitempty" json:"name,omitempty"`
	Description         string   `yaml:"description,omitempty" json:"description,omitempty"`
}

type AddPeerResult struct {
	PublicKey string `yaml:"publicKey" json:"publicKey"`
	Config    string `yaml:"config" json:"config"`
}

type CreateInterfaceRequest struct {
	ServerID   string   `yaml:"serverId" json:"serverId"`
	Name       string   `yaml:"name" json:"name"`
	ListenPort int      `yaml:"listenPort" json:"listenPort"`
	PrivateKey string   `yaml:"privateKey,omitempty" json:"privateKey,omitempty"`
	Endpoint   string   `yaml:"endpoint,omitempty" json:"endpoint,omitempty"`
	AllowedIPs []string `yaml:"allowedIPs,omitempty" json:"allowedIPs,omitempty"`
	Address    string   `yaml:"address,omitempty" json:"address,omitempty"`
}

type KeyPair struct {
	PublicKey  string `yaml:"publicKey" json:"publicKey"`
	PrivateKey string `yaml:"privateKey" json:"privateKey"`
}

type UpdatePeerMetaRequest struct {
	ServerID    string   `yaml:"serverId" json:"serverId"`
	Interface   string   `yaml:"interface" json:"interface"`
	PublicKey   string   `yaml:"publicKey" json:"publicKey"`
	Name        string   `yaml:"name,omitempty" json:"name,omitempty"`
	Description string   `yaml:"description,omitempty" json:"description,omitempty"`
	Endpoint    string   `yaml:"endpoint,omitempty" json:"endpoint,omitempty"`
	AllowedIPs  []string `yaml:"allowedIPs,omitempty" json:"allowedIPs,omitempty"`
	Restart     bool     `yaml:"restart,omitempty" json:"restart,omitempty"`
}

type TestConnectionResult struct {
	Success bool   `yaml:"success" json:"success"`
	Message string `yaml:"message" json:"message"`
}
