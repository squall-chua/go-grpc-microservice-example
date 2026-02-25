package main

import (
	"flag"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func main() {
	secret := flag.String("secret", "super-secret-key", "The HMAC secret key")
	scopes := flag.String("scopes", "read:items write:items", "Space-separated scopes")
	roles := flag.String("roles", "admin", "Comma-separated roles")
	flag.Parse()

	claims := jwt.MapClaims{
		"scope": *scopes,
		"roles": strings.Split(*roles, ","),
		"exp":   time.Now().Add(time.Hour * 24).Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signedToken, err := token.SignedString([]byte(*secret))
	if err != nil {
		log.Fatalf("Error signing token: %v", err)
	}

	fmt.Println(signedToken)
}
