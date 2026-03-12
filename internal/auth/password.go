package auth

import (
	"runtime"

	"golang.org/x/crypto/bcrypt"
)

const bcryptCost = 12

var bcryptSem = make(chan struct{}, runtime.NumCPU())

func HashPassword(plain string) (string, error) {
	bcryptSem <- struct{}{}
	defer func() { <-bcryptSem }()
	b, err := bcrypt.GenerateFromPassword([]byte(plain), bcryptCost)
	return string(b), err
}

func CheckPassword(plain, hashed string) bool {
	bcryptSem <- struct{}{}
	defer func() { <-bcryptSem }()
	return bcrypt.CompareHashAndPassword([]byte(hashed), []byte(plain)) == nil
}
