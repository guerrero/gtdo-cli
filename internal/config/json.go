package config

import (
	"bytes"
	"encoding/json"
	"errors"
)

type filesJSON struct {
	Todo   string `json:"todo"`
	Done   string `json:"done"`
	Report string `json:"report"`
}

type behaviourJSON struct {
	Force               bool   `json:"force"`
	PreserveLineNumbers bool   `json:"preserveLineNumbers"`
	AutoArchive         bool   `json:"autoArchive"`
	DateOnAdd           bool   `json:"dateOnAdd"`
	PriorityOnAdd       string `json:"priorityOnAdd"`
	Verbose             int    `json:"verbose"`
	DefaultAction       string `json:"defaultAction"`
	SourceVar           string `json:"sourceVar"`
	SentenceDelimiters  string `json:"sentenceDelimiters"`
}

type colorsJSON struct {
	PriA string `json:"priA"`
	PriB string `json:"priB"`
	PriC string `json:"priC"`
	PriD string `json:"priD"`
	PriE string `json:"priE"`
	PriF string `json:"priF"`
	PriG string `json:"priG"`
	PriH string `json:"priH"`
	PriI string `json:"priI"`
	PriJ string `json:"priJ"`
	PriK string `json:"priK"`
	PriL string `json:"priL"`
	PriM string `json:"priM"`
	PriN string `json:"priN"`
	PriO string `json:"priO"`
	PriP string `json:"priP"`
	PriQ string `json:"priQ"`
	PriR string `json:"priR"`
	PriS string `json:"priS"`
	PriT string `json:"priT"`
	PriU string `json:"priU"`
	PriV string `json:"priV"`
	PriW string `json:"priW"`
	PriX string `json:"priX"`
	PriY string `json:"priY"`
	PriZ string `json:"priZ"`

	ColorDone    string            `json:"colorDone"`
	ColorProject string            `json:"colorProject"`
	ColorContext string            `json:"colorContext"`
	ColorDate    string            `json:"colorDate"`
	ColorNumber  string            `json:"colorNumber"`
	ColorMeta    string            `json:"colorMeta"`
	Map          map[string]string `json:"map"`
}

type fileConfig struct {
	Dir       string        `json:"dir"`
	Files     filesJSON     `json:"files"`
	Behaviour behaviourJSON `json:"behaviour"`
	Colors    colorsJSON    `json:"colors"`
}

func decodeFileConfig(data []byte, dst *fileConfig) error {
	var raw any
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	if containsJSONNull(raw) {
		return errors.New("null is not allowed in configuration")
	}

	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	return decoder.Decode(dst)
}

func containsJSONNull(value any) bool {
	switch value := value.(type) {
	case nil:
		return true
	case []any:
		for _, item := range value {
			if containsJSONNull(item) {
				return true
			}
		}
	case map[string]any:
		for _, item := range value {
			if containsJSONNull(item) {
				return true
			}
		}
	}
	return false
}
