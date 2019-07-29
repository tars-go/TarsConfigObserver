package TarsConfigObserver

import (
	"bytes"
	"fmt"
	"hash/crc32"
	"path/filepath"
	"strings"
	"time"

	"tars"

	"github.com/spf13/viper"
)

type oneRemoteConfig struct {
	filename    string
	configName  string
	configType  string
	configCRC32 uint32

	viper_inst *viper.Viper
}

// ConfigObserver Taf远程配置监听服务
type ConfigObserver struct {
	ReloadInterval int // 检查远程配置的时间间隔，秒

	rconf          *tars.RConf
	configs        map[string]*oneRemoteConfig
	globalFilename string // 全局文件的文件名，默认是第一个加入的文件
}

// NewObserver 根据服务配置，初始化一个observer (每个App/Server只需要调用1次)
//  reload_interval - 检查远程配置变更的时间，建议值：60 (60秒)。可通过ReloadInterval动态调整
//  path - 传空字符串""则默认写入到`conf/`
func NewObserver(reload_interval int, path string) *ConfigObserver {
	cob := new(ConfigObserver)
	cfg := tars.GetServerConfig()
	if path == "" {
		path = cfg.BasePath + "../conf"
	}

	// init RConf
	cob.rconf = tars.NewRConf(cfg.App, cfg.Server, path)
	cob.configs = make(map[string]*oneRemoteConfig)

	// auto reload
	cob.ReloadInterval = reload_interval
	if cob.ReloadInterval < 1 || cob.ReloadInterval > 3600 {
		cob.ReloadInterval = 60
	}
	go cob.start()

	return cob
}

// AddRemoteConfig 新增一个要监听的远程配置
func (cob *ConfigObserver) AddRemoteConfig(filename string) (vpconf *viper.Viper, err error) {
	configExt := filepath.Ext(filename)
	configName := strings.TrimSuffix(filename, configExt)

	if cob.globalFilename == "" {
		// 第一个加入的配置，作为全局配置直接使用viper的全局对象
		cob.globalFilename = filename
		vpconf = viper.GetViper()
	} else {
		// 之后的对象则创建新的viper实例
		vpconf = viper.New()
	}

	// init oneRemoteConfig and viper
	cob.configs[filename] = &oneRemoteConfig{
		filename:    filename,
		configName:  configName,
		configType:  configExt[1:],
		configCRC32: 0,
		viper_inst:  vpconf,
	}
	vpconf.SetConfigName(configName)
	vpconf.SetConfigType(configExt[1:])

	// reload immediately
	_, err = cob.reloadConfig(cob.configs[filename])
	return vpconf, err
}

func (cob *ConfigObserver) getConfig(filename string) *oneRemoteConfig {
	if conf, ok := cob.configs[filename]; ok {
		return conf
	} else {
		return nil
	}
}

// GetViper 返回目标文件的viper对象
func (cob *ConfigObserver) GetViper(filename string) *viper.Viper {
	conf := cob.getConfig(filename)
	if conf == nil {
		return nil
	}
	return conf.viper_inst
}

// GetCrc32 返回目标文件的CRC32
func (cob *ConfigObserver) GetCRC32(filename string) uint32 {
	conf := cob.getConfig(filename)
	if conf == nil {
		return 0
	}
	return conf.configCRC32
}

// 根据filename读取远程配置
//   return: true|false 是否重载配置，error 是否有错误
func (cob *ConfigObserver) reloadConfig(config *oneRemoteConfig) (bool, error) {
	// logger.Infof("try reloadConfig(%s)", config.filename)
	confBuf, err := cob.rconf.GetConfig(config.filename)
	if err != nil {
		return false, err // RConf API error
	}
	if confBuf == "" {
		return false, fmt.Errorf("Config %s is empty", config.filename)
	}

	// 检查crc32是否一致，不一致则触发reload
	newCRC32 := crc32.ChecksumIEEE([]byte(confBuf))
	if config.configCRC32 != newCRC32 {
		// logger.Infof("real reload config: %s", confBuf)
		config.configCRC32 = newCRC32
		// 重载viper的配置数据
		return true, config.viper_inst.ReadConfig(bytes.NewBuffer([]byte(confBuf)))
	}
	return false, nil
}

// 开始监听配置变更
func (cob *ConfigObserver) start() {
	for {
		time.Sleep(time.Duration(cob.ReloadInterval) * time.Second)
		// 定期检查指定远程配置的变更
		for _, conf := range cob.configs {
			cob.reloadConfig(conf)
		}
	}
}
