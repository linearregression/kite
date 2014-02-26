// Package regserv implements a registration server kite. Users can register
// to a kite infrastructure by running "kite register" command.
package regserv

import (
	"errors"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/koding/kite"
	"github.com/koding/kite/config"
	"github.com/koding/kite/server"
	"github.com/nu7hatch/gouuid"
)

const Version = "0.0.2"

// RegServ is a registration kite. Users can register their machines by
// running "kite register" command.
type RegServ struct {
	Server       *server.Server
	Authenticate func(r *kite.Request) (username string, err error)
	publicKey    string
	privateKey   string
}

func New(conf *config.Config, pubKey, privKey string) *RegServ {
	k := kite.New("regserv", Version)
	r := &RegServ{
		Server:       server.New(k),
		Authenticate: AskUsernameOnly,
		publicKey:    pubKey,
		privateKey:   privKey,
	}
	k.HandleFunc("register", r.handleRegister)
	return r
}

func (s *RegServ) Run() {
	s.Server.Run()
	// fmt.Println("Users can register with the following command:")
	// fmt.Printf("kite register -to '%s'\n", s.kite.URL.String())
}

// // RegisterSelf registers this host and writes a key to ~/.kite/kite.key
// func (s *RegServ) RegisterSelf() error {
// 	hostname, err := os.Hostname()
// 	if err != nil {
// 		return err
// 	}
// 	key, err := s.register(s.backend.Username(), hostname)
// 	if err != nil {
// 		return err
// 	}
// 	return kitekey.Write(key)
// }

func (s *RegServ) handleRegister(r *kite.Request) (interface{}, error) {
	var args struct {
		Hostname string
	}
	r.Args.One().MustUnmarshal(&args)

	username, err := s.Authenticate(r)
	if err != nil {
		return nil, errors.New("cannot authenticate user")
	}

	return s.register(username, args.Hostname)
}

func (s *RegServ) register(username, hostname string) (kiteKey string, err error) {
	tknID, err := uuid.NewV4()
	if err != nil {
		return "", errors.New("cannot generate a token")
	}

	token := jwt.New(jwt.GetSigningMethod("RS256"))

	token.Claims = map[string]interface{}{
		"iss":        s.Server.Kite.Kite().Username, // Issuer
		"sub":        username,                      // Subject
		"aud":        hostname,                      // Hostname of registered machine
		"iat":        time.Now().UTC().Unix(),       // Issued At
		"jti":        tknID.String(),                // JWT ID
		"kontrolURL": s.Server.Config.Username,      // Kontrol URL
		"kontrolKey": s.publicKey,                   // Public key of kontrol
	}

	return token.SignedString([]byte(s.privateKey))
}

// AskUsernameOnly is a function for authentication user. It asks for only
// username. You should probably not use this and authenticate users be
// asking a password or something different.
func AskUsernameOnly(r *kite.Request) (string, error) {
	result, err := r.RemoteKite.TellWithTimeout("prompt", 10*time.Minute, "Enter username: ")
	if err != nil {
		return "", err
	}
	return result.MustString(), nil
}
