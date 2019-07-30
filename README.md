# TarsConfigObserver
Make life easier, read tars remote config with spf13/viper

## Usage

First, create an observer and add one or more remote config names:

~~~golang
    // Init remote config
    rconf_obs := TarsConfigObserver.NewObserver(60, "")
    _, err := rconf_obs.AddRemoteConfig("config.yaml")
    if err != nil {
        return err
    }
~~~

Second, use viper like local config:

~~~golang
    // any file in project
    viper.GetString("foo.bar")
~~~

## Mutil configs

~~~golang
    rconf_obs := TarsConfigObserver.NewObserver(60, "")
    _, err := rconf_obs.AddRemoteConfig("config.yaml")
    if err != nil {
        return err
    }

    json_viper, err := rconf_obs.AddRemoteConfig("second_config.json")
    if err != nil {
        return err
    }

    // read second configs
    json_viper.GetString("key")

    // or get from GetViper()
    other_viper := rconf_obs.GetViper("second_config.json")
    other_viper.GetString("key")
~~~
