package handlers

import (
	"log"
	"net/http"
)

func respondWithError(w http.ResponseWriter, status int, userMsg, logMsg string, err error) {
	if err != nil {
		if logMsg == "" {
			logMsg = userMsg
		}
		log.Printf("%s: %v", logMsg, err)
	}

	http.Error(w, userMsg, status)
}
