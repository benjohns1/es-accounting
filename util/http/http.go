package http

import (
	"fmt"
	"log"
	"net/http"
)

func WriteLogResponse(w http.ResponseWriter, status int, format string, a ...interface{}) {
	msg := fmt.Sprintf(format, a...)
	log.Println(msg)
	w.WriteHeader(status)
	w.Write([]byte(msg))
}

func WriteErrJSONResponse(w http.ResponseWriter, status int, err error) {
	WriteErrStrJSONResponse(w, status, err.Error())
}

func WriteErrStrJSONResponse(w http.ResponseWriter, status int, msg string) {
	log.Println(msg)
	w.Header().Add("Content-Type", "application/json")
	w.WriteHeader(status)
	w.Write([]byte(fmt.Sprintf(`{"error":"%s"}`, msg)))
}
