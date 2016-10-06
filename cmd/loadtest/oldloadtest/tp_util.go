package oldloadtest

import (
	"crypto/rand"
	"fmt"
	mrand "math/rand"
)

// GeneratePlatformFirst based on base string
func GeneratePlatformFirst(id int) string {
	return fmt.Sprint(Config.UserFirst, id)
}

// GeneratePlatformEmail based on base and domain string
func GeneratePlatformEmail(id int) string {
	return fmt.Sprint(Config.UserEmail, id, Config.EmailDomain)
}

// GeneratePlatformLast just based on const for now
func GeneratePlatformLast() string {
	return Config.UserLast
}

// GeneratePlatformUsername based on base string
func GeneratePlatformUsername(id int) string {
	return fmt.Sprint(Config.UserName, id)
}

// GeneratePlatformPass based on base string
func GeneratePlatformPass(id int) string {
	return fmt.Sprint(Config.UserPassword, id)
}

// RandomChoice will pick a number out of 100 and return true if it's below the percent
func RandomChoice(successPercent int) bool {
	random := mrand.Intn(100)
	return random < successPercent
}

// GenerateUUID creates a psuedo UUID
func GenerateUUID() (uuid string) {
	b := make([]byte, 16)
	_, err := rand.Read(b)
	if err != nil {
		fmt.Println("Error: ", err)
		return
	}
	uuid = fmt.Sprintf("%X-%X-%X-%X-%X", b[0:4], b[4:6], b[6:8], b[8:10], b[10:])
	return
}
