package main

import (
    "errors"
)

var (
    ErrAuthMacIdMismatch = errors.New("mac/id mismatch")
    ErrAuthEmulatorId = errors.New("emulator (id)")
    ErrAuthEmulatorMac = errors.New("emulator (mac)")
    ErrAuthUnderage = errors.New("user age<13")
    ErrAuthUnknownError = errors.New("unknown error")
    ErrIpBan = errors.New("user is ip banned")
    ErrIdBan = errors.New("user is id banned")
)
