// Copyright (c) 2025 Proton AG
//
// This file is part of Proton Mail Bridge.
//
// Proton Mail Bridge is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// Proton Mail Bridge is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with Proton Mail Bridge. If not, see <https://www.gnu.org/licenses/>.

package storage

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/sirupsen/logrus"
)

type JSONFile[T any] struct {
	fileName string
	fileType string
}

func NewJSONStorageFile[T any](fileName, fileType string) *JSONFile[T] {
	return &JSONFile[T]{
		fileName: fileName,
		fileType: fileType,
	}
}

func (jf *JSONFile[T]) Load(vaultDir string) (T, error) {
	var result T
	path := filepath.Join(vaultDir, jf.fileName)
	bytes, err := os.ReadFile(path) //nolint:gosec
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			logrus.
				WithFields(logrus.Fields{"pkg": "vault", "path": path}).
				Tracef("%s file does not exists, default values will be used", jf.fileType)
			return result, nil
		}
		return result, err
	}

	if err := json.Unmarshal(bytes, &result); err != nil {
		return result, fmt.Errorf("%s file has invalid data: %w", jf.fileType, err)
	}

	return result, nil
}

func (jf *JSONFile[T]) Save(vaultDir string, data T) error {
	bytes, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}

	if err = os.MkdirAll(vaultDir, 0o700); err != nil {
		return err
	}

	path := filepath.Join(vaultDir, jf.fileName)
	return os.WriteFile(path, bytes, 0o600)
}
