package cmds

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"

	"github.com/manuel/wesen/tuplespace/internal/types"
)

func LoadTuple(path string) (types.Tuple, error) {
	body, err := os.ReadFile(path)
	if err != nil {
		return types.Tuple{}, fmt.Errorf("read tuple file: %w", err)
	}
	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.UseNumber()

	var tuple types.Tuple
	if err := decoder.Decode(&tuple); err != nil {
		return types.Tuple{}, fmt.Errorf("decode tuple file: %w", err)
	}
	normalized, err := types.NormalizeTuple(tuple)
	if err != nil {
		return types.Tuple{}, fmt.Errorf("normalize tuple file: %w", err)
	}
	return normalized, nil
}

func LoadTemplate(path string) (types.Template, error) {
	body, err := os.ReadFile(path)
	if err != nil {
		return types.Template{}, fmt.Errorf("read template file: %w", err)
	}
	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.UseNumber()

	var template types.Template
	if err := decoder.Decode(&template); err != nil {
		return types.Template{}, fmt.Errorf("decode template file: %w", err)
	}
	normalized, err := types.NormalizeTemplate(template)
	if err != nil {
		return types.Template{}, fmt.Errorf("normalize template file: %w", err)
	}
	return normalized, nil
}
