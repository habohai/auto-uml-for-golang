package config

// Config Application config
type Config struct {
	RunTime  *RunTimeConfig
	GoEnv    *GoEnvConfig
	CodeArgs *CodeArgsConfig
}

// RunTimeConfig config
type RunTimeConfig struct {
	// Address for grpc service.
	Log *LogConfig
}

// LogConfig config
type LogConfig struct {
	SavePath string
	SaveName string
	FileExt  string
}

// GoEnvConfig config
type GoEnvConfig struct {
	GoPath string
}

// CodeArgsConfig config
type CodeArgsConfig struct {
	CodePath   string
	OutputPath string
}
