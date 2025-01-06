package main

import (
	"errors"
)

var (
    ErrAuthMacFsidMismatch = errors.New("mac/fsid mismatch")
    ErrAuthEmulatorId = errors.New("emulator (fsid)")
    ErrAuthEmulatorMac = errors.New("emulator (mac)")
    ErrAuthUnderage = errors.New("user age<13")
    ErrAuthUnknownError = errors.New("unknown error")
    ErrIpBan = errors.New("user is ip banned")
    ErrFsidBan = errors.New("user is fsid banned")
    ErrAlreadyBanned = errors.New("user is already banned")
    ErrInvalidDbType = errors.New("invalid db type in config")
    ErrNoSid = errors.New("invalid sid")
    ErrNoUser = errors.New("no user exists with this fsid")
    ErrMovieExists = errors.New("flipnote already exists on server")
)
