package config

import (
	"errors"
	"fmt"
	"math/big"
	"sort"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cast"
	"github.com/spf13/viper"
)

const (
	AgentContractAddressKey            = "AGENT_CONTRACT_ADDRESS"
	MultiPartyEscrowContractAddressKey = "MULTI_PARTY_ESCROW_CONTRACT_ADDRESS"
	AutoSSLDomainKey                   = "AUTO_SSL_DOMAIN"
	AutoSSLCacheDirKey                 = "AUTO_SSL_CACHE_DIR"
	BlockchainEnabledKey               = "BLOCKCHAIN_ENABLED"
	ConfigPathKey                      = "CONFIG_PATH"
	DaemonListeningPortKey             = "DAEMON_LISTENING_PORT"
	DaemonTypeKey                      = "DAEMON_TYPE"
	DbPathKey                          = "DB_PATH"
	EthereumJsonRpcEndpointKey         = "ETHEREUM_JSON_RPC_ENDPOINT"
	ExecutablePathKey                  = "EXECUTABLE_PATH"
	HdwalletIndexKey                   = "HDWALLET_INDEX"
	HdwalletMnemonicKey                = "HDWALLET_MNEMONIC"
	PricePerCallKey                    = "PRICE_PER_CALL"
	LogKey                             = "LOG"
	PassthroughEnabledKey              = "PASSTHROUGH_ENABLED"
	PassthroughEndpointKey             = "PASSTHROUGH_ENDPOINT"
	PollSleepKey                       = "POLL_SLEEP"
	PrivateKeyKey                      = "PRIVATE_KEY"
	ServiceTypeKey                     = "SERVICE_TYPE"
	SSLCertPathKey                     = "SSL_CERT"
	SSLKeyPathKey                      = "SSL_KEY"
	WireEncodingKey                    = "WIRE_ENCODING"
	ReplicaGroupIDKey                  = "REPLICA_GROUP_ID" // TODO: read replica group id from IPFS metadata
	PaymentChannelStorageTypeKey       = "PAYMENT_CHANNEL_STORAGE_TYPE"
	PaymentChannelStorageClientKey     = "PAYMENT_CHANNEL_STORAGE_CLIENT"
	PaymentChannelStorageServerKey     = "PAYMENT_CHANNEL_STORAGE_SERVER"

	defaultConfigJson string = `
{
	"auto_ssl_cache_dir": ".certs",
	"auto_ssl_domain": "",
	"blockchain_enabled": true,
	"daemon_listening_port": 5000,
	"daemon_type": "grpc",
	"db_path": "snetd.db",
	"ethereum_json_rpc_endpoint": "http://127.0.0.1:8545",
	"hdwallet_index": 0,
	"hdwallet_mnemonic": "",
	"price_per_call": 10,
	"passthrough_enabled": false,
	"poll_sleep": "5s",
	"service_type": "grpc",
	"ssl_cert": "",
	"ssl_key": "",
	"wire_encoding": "proto",
	"log":  {
		"level": "info",
		"timezone": "UTC",
		"formatter": {
			"type": "text"
		},
		"output": {
			"type": "file",
			"file_pattern": "./snet-daemon.%Y%m%d.log",
			"current_link": "./snet-daemon.log",
			"rotation_time_in_sec": 86400,
			"max_age_in_sec": 604800,
			"rotation_count": 0
		},
		"hooks": []
	},
	"replica_group_id": "0",
	"payment_channel_storage_type": "memory",
	"payment_channel_storage_client": {
		"connection_timeout": 5000,
		"request_timeout": 3000,
		"endpoints": ["http://127.0.0.1:2379"]
	},
	"payment_channel_storage_server": {
		"id": "storage-1",
		"scheme": "http",
		"host" : "127.0.0.1",
		"client_port": 2379,
		"peer_port": 2380,
		"token": "unique-token",
		"cluster": "storage-1=http://127.0.0.1:2380",
		"enabled": false
	}
}
`
)

var vip *viper.Viper

func init() {
	var err error

	vip = viper.New()
	vip.SetEnvPrefix("SNET")
	vip.AutomaticEnv()

	var defaults = viper.New()
	err = ReadConfigFromJsonString(defaults, defaultConfigJson)
	if err != nil {
		panic(fmt.Sprintf("Cannot load default config: %v", err))
	}
	SetDefaultFromConfig(vip, defaults)

	vip.AddConfigPath(".")
}

// ReadConfigFromJsonString function reads settigs from json string to the
// config instance. String should contain valid JSON config.
func ReadConfigFromJsonString(config *viper.Viper, json string) error {
	config.SetConfigType("json")
	return config.ReadConfig(strings.NewReader(json))
}

// SetDefaultFromConfig sets all settings from defaults as default values to
// the config.
func SetDefaultFromConfig(config *viper.Viper, defaults *viper.Viper) {
	for key, value := range defaults.AllSettings() {
		config.SetDefault(key, value)
	}
}

func Vip() *viper.Viper {
	return vip
}

func Validate() error {
	switch dType := vip.GetString(DaemonTypeKey); dType {
	case "grpc":
		switch sType := vip.GetString(ServiceTypeKey); sType {
		case "grpc":
		case "jsonrpc":
		case "process":
			if vip.GetString(ExecutablePathKey) == "" {
				return errors.New("EXECUTABLE required with SERVICE_TYPE 'process'")
			}
		default:
			return fmt.Errorf("unrecognized SERVICE_TYPE '%+v'", sType)
		}

		switch enc := vip.GetString(WireEncodingKey); enc {
		case "proto":
		case "json":
		default:
			return fmt.Errorf("unrecognized WIRE_ENCODING '%+v'", enc)
		}
	case "http":
	default:
		return fmt.Errorf("unrecognized DAEMON_TYPE '%+v'", dType)
	}

	if vip.GetBool(BlockchainEnabledKey) {
		if vip.GetString(PrivateKeyKey) == "" && vip.GetString(HdwalletMnemonicKey) == "" {
			return errors.New("either PRIVATE_KEY or HDWALLET_MNEMONIC are required")
		}
	}

	certPath, keyPath := vip.GetString(SSLCertPathKey), vip.GetString(SSLKeyPathKey)
	if (certPath != "" && keyPath == "") || (certPath == "" && keyPath != "") {
		return errors.New("SSL requires both key and certificate when enabled")
	}

	return nil
}

func LoadConfig(configFile string) error {
	vip.SetConfigFile(configFile)
	return vip.ReadInConfig()
}

func WriteConfig(configFile string) error {
	vip.SetConfigFile(configFile)
	return vip.WriteConfig()
}

func GetString(key string) string {
	return vip.GetString(key)
}

func GetInt(key string) int {
	return vip.GetInt(key)
}

func GetBigInt(key string) *big.Int {
	return big.NewInt(int64(vip.GetInt(key)))
}

func GetDuration(key string) time.Duration {
	return vip.GetDuration(key)
}

func GetBool(key string) bool {
	return vip.GetBool(key)
}

func GetDefaultConfJSON() string {
	return defaultConfigJson
}

// SubWithDefault returns sub-config by keys including configuration defaults
// values. It returns nil if no such key. It is analog of the viper.Sub()
// function. This is workaround for the issue
// https://github.com/spf13/viper/issues/559
func SubWithDefault(config *viper.Viper, key string) *viper.Viper {
	var allSettingsByKey, ok = config.AllSettings()[strings.ToLower(key)]
	if !ok {
		return nil
	}

	var subMap = cast.ToStringMap(allSettingsByKey)
	var sub = viper.New()
	for subKey, value := range subMap {
		sub.Set(subKey, value)
	}

	return sub
}

var hiddenKeys = map[string]bool{
	strings.ToUpper(PrivateKeyKey):       true,
	strings.ToUpper(HdwalletMnemonicKey): true,
}

func LogConfig() {
	log.Info("Final configuration:")
	keys := vip.AllKeys()
	sort.Strings(keys)
	for _, key := range keys {
		if hiddenKeys[strings.ToUpper(key)] {
			log.Infof("%v: ***", key)
		} else {
			log.Infof("%v: %v", key, vip.Get(key))
		}
	}
}
