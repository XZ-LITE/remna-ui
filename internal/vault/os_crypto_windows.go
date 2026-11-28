//go:build windows

package vault

import (
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"syscall"
	"unsafe"
)

var (
	modcrypt32             = syscall.NewLazyDLL("crypt32.dll")
	procCryptProtectData   = modcrypt32.NewProc("CryptProtectData")
	procCryptUnprotectData = modcrypt32.NewProc("CryptUnprotectData")
)

type dataBlob struct {
	cbData uint32
	pbData *byte
}

func (m *Manager) getOSProtectionBackend() string {
	return "dpapi"
}

func (m *Manager) encryptOSSecret(secret string) (string, error) {
	return encryptDPAPI(secret)
}

func (m *Manager) decryptOSSecret(encrypted string) (string, error) {
	if strings.HasPrefix(encrypted, prefixDPAPI) {
		return decryptDPAPI(encrypted)
	}
	return encrypted, nil
}

func encryptDPAPI(secret string) (string, error) {
	if secret == "" || isEncryptedSecret(secret) {
		return secret, nil
	}
	bytesData := []byte(secret)
	var inBlob, outBlob dataBlob
	inBlob.cbData = uint32(len(bytesData))
	if len(bytesData) > 0 {
		inBlob.pbData = &bytesData[0]
	}
	r, _, callErr := procCryptProtectData.Call(
		uintptr(unsafe.Pointer(&inBlob)),
		0,
		0,
		0,
		0,
		0x01, // CRYPTPROTECT_UI_FORBIDDEN
		uintptr(unsafe.Pointer(&outBlob)),
	)
	if r == 0 || outBlob.pbData == nil {
		return "", fmt.Errorf("ошибка DPAPI шифрования: %v", callErr)
	}
	defer syscall.LocalFree(syscall.Handle(unsafe.Pointer(outBlob.pbData)))
	outBytes := make([]byte, outBlob.cbData)
	for i := uint32(0); i < outBlob.cbData; i++ {
		outBytes[i] = *(*byte)(unsafe.Pointer(uintptr(unsafe.Pointer(outBlob.pbData)) + uintptr(i)))
	}
	return prefixDPAPI + base64.StdEncoding.EncodeToString(outBytes), nil
}

func decryptDPAPI(encrypted string) (string, error) {
	if !strings.HasPrefix(encrypted, prefixDPAPI) {
		return encrypted, nil
	}
	rawB64 := strings.TrimPrefix(encrypted, prefixDPAPI)
	data, err := base64.StdEncoding.DecodeString(rawB64)
	if err != nil || len(data) == 0 {
		return "", errors.New("некорректные данные DPAPI")
	}
	var inBlob, outBlob dataBlob
	inBlob.cbData = uint32(len(data))
	inBlob.pbData = &data[0]
	r, _, callErr := procCryptUnprotectData.Call(
		uintptr(unsafe.Pointer(&inBlob)),
		0,
		0,
		0,
		0,
		0x01, // CRYPTPROTECT_UI_FORBIDDEN
		uintptr(unsafe.Pointer(&outBlob)),
	)
	if r == 0 || outBlob.pbData == nil {
		return "", fmt.Errorf("ошибка DPAPI расшифровки: %v", callErr)
	}
	defer syscall.LocalFree(syscall.Handle(unsafe.Pointer(outBlob.pbData)))
	outBytes := make([]byte, outBlob.cbData)
	for i := uint32(0); i < outBlob.cbData; i++ {
		outBytes[i] = *(*byte)(unsafe.Pointer(uintptr(unsafe.Pointer(outBlob.pbData)) + uintptr(i)))
	}
	return string(outBytes), nil
}
