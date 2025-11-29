package config

// Listener default config
type Listener struct {
	Listen string `yaml:"listen" default:"0.0.0.0"`
	Port   int    `yaml:"port" default:"8081"`
}

type SQLConfig struct {
	Enable          bool   `yaml:"enable" default:"false" desc:"config:sql:enable"`
	Driver          string `yaml:"driver" default:"" desc:"config:sql:driver"`
	Host            string `yaml:"host" default:"127.0.0.1" desc:"config:sql:host"`
	Port            int    `yaml:"port" default:"3306" desc:"config:sql:port"`
	Username        string `yaml:"username" default:"root"  desc:"config:sql:username"`
	Password        string `yaml:"password" default:"root" desc:"config:sql:password"`
	Database        string `yaml:"database" default:"database" desc:"config:sql:database"`
	Options         string `yaml:"options" default:"" desc:"config:sql:options"`
	Connection      string `yaml:"connection" default:"" desc:"config:sql:connection"`
	AutoReconnect   bool   `yaml:"autoReconnect" default:"false"  desc:"config:sql:autoReconnect"`
	StartInterval   int    `yaml:"startInterval" default:"2"  desc:"config:sql:startInterval"`
	MaxError        int    `yaml:"maxError" default:"5"  desc:"config:sql:maxError"`
	CustomPool      bool   `yaml:"customPool" default:"false"  desc:"config:sql:customPool"`
	MaxConn         int    `yaml:"maxConn" default:"5"  desc:"config:sql:maxConn"`
	MaxIdle         int    `yaml:"maxIdle" default:"5"  desc:"config:sql:maxIdle"`
	LifeTime        int    `yaml:"lifeTime" default:"5"  desc:"config:sql:lifeTime"`
	MultiStatements bool   `yaml:"multiStatements" default:"false"  desc:"config:sql:multiStatements"`
	UseMock         bool   `yaml:"useMock" default:"false"  desc:"config:sql:useMock"`
}

type RabbitMQConfig struct {
	Enable              bool   `yaml:"enable" default:"false" desc:"config:rabbitmq:enable"`
	Host                string `yaml:"host" default:"127.0.0.1" desc:"config:rabbitmq:host"`
	Port                int    `yaml:"port" default:"5672" desc:"config:rabbitmq:port"`
	Username            string `yaml:"username" default:"guest"  desc:"config:rabbitmq:username"`
	Password            string `yaml:"password" default:"guest" desc:"config:rabbitmq:password"`
	ReconnectDuration   int    `yaml:"reconnectDuration" default:"5" desc:"config:rabbitmq:reconnectDuration"`
	DedicatedConnection bool   `yaml:"dedicatedConnection" default:"false" desc:"config:rabbitmq:dedicatedConnection"`
	UseMock             bool   `yaml:"useMock" default:"false"  desc:"config:useMock"`
}

type RedisConfig struct {
	Enable        bool   `yaml:"enable" default:"false" desc:"config:redis:enable"`
	Host          string `yaml:"host" default:"127.0.0.1" desc:"config:redis:host"`
	Port          int    `yaml:"port" default:"6379" desc:"config:redis:port"`
	Password      string `yaml:"password" default:"" desc:"config:redis:password"`
	Pool          int    `yaml:"pool" default:"10" desc:"config:redis:pool"`
	AutoReconnect bool   `yaml:"autoReconnect" default:"false"  desc:"config:redis:autoReconnect"`
	StartInterval int    `yaml:"startInterval" default:"2"  desc:"config:redis:startInterval"`
	MaxError      int    `yaml:"maxError" default:"5"  desc:"config:redis:maxError"`
	PoolSize      int    `yaml:"poolSize" default:"30" desc:"config:redis:poolSize"`
	PoolTimeout   int    `yaml:"poolTimeout" default:"30" desc:"config:redis:poolTimeout"`
	MinIdleConn   int    `yaml:"minIdleConn" default:"7" desc:"config:redis:minIdleConn"`
	MaxIdleConn   int    `yaml:"maxIdleConn" default:"15" desc:"config:redis:maxIdleConn"`
	ConnMaxLife   int    `yaml:"connMaxLife" default:"600" desc:"config:redis:connMaxLife"`
	UseMock       bool   `yaml:"useMock" default:"false"  desc:"config:useMock"`
}

type KafkaConfig struct {
	Enable           bool   `yaml:"enable" default:"false" desc:"config:kafka:enable"`
	Host             string `yaml:"host" default:"127.0.0.1:9092" desc:"config:kafka:host"`
	Registry         string `yaml:"registry" default:"" desc:"config:kafka:registry"`
	Username         string `yaml:"username" default:""  desc:"config:kafka:username"`
	Password         string `yaml:"password" default:"" desc:"config:kafka:password"`
	SecurityProtocol string `yaml:"securityProtocol" default:"SASL_SSL"  desc:"config:kafka:securityProtocol"`
	Mechanisms       string `yaml:"mechanisms" default:"PLAIN"  desc:"config:kafka:mechanisms"`
	UseMock          bool   `yaml:"useMock" default:"false"  desc:"config:useMock"`
	Debug            string `yaml:"debug" default:"consumer"  desc:"config:kafka:debug"`
}

type JWTConfig struct {
	Access         string `yaml:"access" default:"random"`
	Refresh        string `yaml:"refresh" default:"random"`
	ExpiredAccess  int    `yaml:"expiredAccess" default:"30"`
	ExpiredRefresh int    `yaml:"expiredRefresh" default:"24"`
	UseMock        bool   `yaml:"useMock" default:"false"  desc:"config:useMock"`
}
