package credentials

import (
	"crypto/rand"
	"math/big"
)

// Word lists for generating kid-friendly usernames
var adjectives = []string{
	"happy", "sunny", "brave", "bright", "cool", "swift", "clever", "jolly",
	"mighty", "super", "star", "wild", "funny", "lucky", "magic", "bouncy",
	"cheerful", "daring", "eager", "flying", "gentle", "hyper", "jazzy", "kindly",
	"lively", "merry", "noble", "perky", "quick", "royal", "snappy", "turbo",
	"zippy", "awesome", "bold", "cosmic", "dynamic", "epic", "fantastic", "groovy",
}

var nouns = []string{
	"dragon", "tiger", "eagle", "dolphin", "panda", "lion", "wolf", "bear",
	"fox", "hawk", "shark", "phoenix", "unicorn", "rocket", "ninja", "wizard",
	"knight", "pirate", "robot", "astronaut", "hero", "champion", "explorer", "ranger",
	"warrior", "captain", "genius", "comet", "thunder", "lightning", "tornado", "blizzard",
	"flame", "storm", "shadow", "spirit", "ghost", "monster", "alien", "racer",
}

// GenerateKidUsername generates a random username in the format "adjective-noun"
func GenerateKidUsername() (string, error) {
	adjective, err := randomElement(adjectives)
	if err != nil {
		return "", err
	}

	noun, err := randomElement(nouns)
	if err != nil {
		return "", err
	}

	return adjective + "-" + noun, nil
}

// GenerateKidPassword generates a random 4-character password using letters and numbers
func GenerateKidPassword() (string, error) {
	chars := "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	password := make([]byte, 4)

	for i := 0; i < 4; i++ {
		num, err := rand.Int(rand.Reader, big.NewInt(int64(len(chars))))
		if err != nil {
			return "", err
		}
		password[i] = chars[num.Int64()]
	}

	return string(password), nil
}

// randomElement picks a random element from a string slice
func randomElement(slice []string) (string, error) {
	if len(slice) == 0 {
		return "", nil
	}

	num, err := rand.Int(rand.Reader, big.NewInt(int64(len(slice))))
	if err != nil {
		return "", err
	}

	return slice[num.Int64()], nil
}
