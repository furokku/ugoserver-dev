package main

import (
	"errors"
)

var (
    ErrAuthMacFsidMismatch = errors.New("mac/fsid mismatch")
    ErrAuthEmulatorId = errors.New("emulator (fsid)")
    ErrAuthEmulatorMac = errors.New("emulator (mac)")
    ErrAuthUnderage = errors.New("user age<13")
    ErrAuthInvalidRegion = errors.New("invalid region")
    ErrAuthUnknownError = errors.New("unknown error")
    ErrIpBan = errors.New("user is ip banned")
    ErrFsidBan = errors.New("user is fsid banned")
    ErrAlreadyBanned = errors.New("user is already banned")
    ErrNoSid = errors.New("invalid sid")
    ErrNoUser = errors.New("no user exists with this fsid")
    ErrNoMovie = errors.New("movie with such id does not exist")
    ErrMovieExists = errors.New("flipnote already exists on server")
)
