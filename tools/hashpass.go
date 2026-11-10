
package main

import (
	"fmt"
	"os"

	"golang.org/x/crypto/bcrypt"
)

func main() {
	password := os.Getenv("STAFF_PASSWORD")

	if password == "" {
		fmt.Println("Password was not provided.")
		return
	}

	hash, err := bcrypt.GenerateFromPassword(
		[]byte(password),
		bcrypt.DefaultCost,
	)

	if err != nil {
		fmt.Println("Could not generate password hash:", err)
		return
	}

	fmt.Println(string(hash))
}
