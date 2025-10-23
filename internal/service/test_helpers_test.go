package service

import "github.com/rs/zerolog"

func testLogger() zerolog.Logger {
	return zerolog.New(zerolog.NewConsoleWriter())
}
