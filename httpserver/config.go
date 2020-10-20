package httpserver

type HttpConfig struct {
	Addr       string `json:"addr" toml:"addr"`
	IsTLS      bool   `json:"is_tls" toml:"is_tls"`
	TLSCrtFile string `json:"tls_crt_file" toml:"tls_crt_file"`
	TLSKeyFile string `json:"tls_key_file" toml:"tls_key_file"`
}
