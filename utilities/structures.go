package utilities

var ConfigMagic = []byte("WcCf")

type ConfigFormat struct {
	Magic             [4]byte
	Version           int32
	FriendCode        int64
	AmountOfCreations int32
	HasRegistered     int32
	MailDomain        [64]byte
	Passwd            [32]byte
	Mlchkid           [36]byte
	AccountURL        [128]byte
	CheckURL          [128]byte
	ReceiveURL        [128]byte
	DeleteURL         [128]byte
	SendURL           [128]byte
	_                 [220]byte // Most likely reserved.
	TitleBooting      int32
	Checksum          [4]byte
}

// Config structure for `config.json`.
type Config struct {
	Port            int
	Host            string
	Username        string
	Password        string
	DBName          string
	Interval        int
	BindTo          string
	SendGridKey     string
	SendGridDomain  string
	Debug           bool
	PatchBaseDomain string
	RavenDSN        string
}