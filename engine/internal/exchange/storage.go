package exchange

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"os"
	"sort"
	"strings"
)

func digest(parts ...[]byte) string {
	hash := sha256.New()
	for _, part := range parts {
		_, _ = hash.Write(part)
	}
	return hex.EncodeToString(hash.Sum(nil))
}

func validDigest(value string) bool {
	decoded, err := hex.DecodeString(value)
	return err == nil && len(decoded) == sha256.Size && hex.EncodeToString(decoded) == value
}

func strictCanonical(raw []byte, value any) error {
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if decoder.Decode(value) != nil || decoder.Decode(&struct{}{}) != io.EOF {
		return errors.New("strict JSON")
	}
	canonical, _ := json.Marshal(value)
	if !bytes.Equal(canonical, raw) {
		return errors.New("noncanonical JSON")
	}
	return nil
}

func (s *Service) publish(directory, name, identity string, raw []byte) error {
	temp := directory + "/." + identity + ".tmp"
	file, err := s.root.OpenFile(temp, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return &Error{"STORAGE_ERROR", "exchange temporary create"}
	}
	defer s.root.Remove(temp)
	if _, err = file.Write(raw); err == nil {
		err = file.Sync()
	}
	closeErr := file.Close()
	if err == nil {
		err = closeErr
	}
	if err != nil {
		return &Error{"STORAGE_ERROR", "exchange temporary write"}
	}
	if err = s.root.Link(temp, name); err != nil {
		return &Error{"STORAGE_ERROR", "exchange no-replace publication"}
	}
	if err = s.root.Remove(temp); err != nil {
		return &Error{"STORAGE_ERROR", "exchange temporary cleanup"}
	}
	return s.sync(directory)
}

func (s *Service) readRegular(name string, limit int64) ([]byte, error) {
	info, err := s.root.Lstat(name)
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() || info.Mode().Perm()&0077 != 0 || info.Size() < 1 || info.Size() > limit {
		return nil, &Error{"UNSAFE_STORAGE", "exchange file"}
	}
	file, err := s.root.Open(name)
	if err != nil {
		return nil, &Error{"STORAGE_ERROR", "exchange file open"}
	}
	raw, readErr := io.ReadAll(io.LimitReader(file, limit+1))
	closeErr := file.Close()
	if readErr != nil || closeErr != nil || int64(len(raw)) > limit {
		return nil, &Error{"STORAGE_ERROR", "exchange file read"}
	}
	return raw, nil
}

func (s *Service) readDirectory(name string, limit int) ([]os.DirEntry, error) {
	directory, err := s.root.Open(name)
	if err != nil {
		return nil, &Error{"STORAGE_ERROR", "exchange directory"}
	}
	defer directory.Close()
	entries, err := directory.ReadDir(limit)
	if err != nil && err != io.EOF {
		return nil, &Error{"STORAGE_ERROR", "exchange directory"}
	}
	if len(entries) == limit && err == nil {
		return nil, &Error{"LIMIT_EXCEEDED", "exchange directory entries"}
	}
	return entries, nil
}

func (s *Service) sync(name string) error {
	directory, err := s.root.Open(name)
	if err != nil {
		return &Error{"STORAGE_ERROR", "exchange directory sync"}
	}
	err = directory.Sync()
	directory.Close()
	if err != nil {
		return &Error{"STORAGE_ERROR", "exchange directory sync"}
	}
	return nil
}

func canonicalStrings(values []string) ([]string, bool) {
	result := append([]string(nil), values...)
	sort.Strings(result)
	for index, value := range result {
		if value == "" || strings.ContainsRune(value, 0) || index > 0 && result[index-1] == value {
			return nil, false
		}
	}
	return result, true
}
