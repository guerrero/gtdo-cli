package config

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
)

type filesJSON struct {
	Todo   string `json:"todo"`
	Done   string `json:"done"`
	Report string `json:"report"`
}

type behaviourJSON struct {
	Force               bool     `json:"force"`
	PreserveLineNumbers bool     `json:"preserveLineNumbers"`
	AutoArchive         bool     `json:"autoArchive"`
	EnableUUID          bool     `json:"enableUUID"`
	PriorityOnAdd       string   `json:"priorityOnAdd"`
	Verbose             int      `json:"verbose"`
	DefaultAction       string   `json:"defaultAction"`
	SourceVar           string   `json:"sourceVar"`
	SentenceDelimiters  string   `json:"sentenceDelimiters"`
	TaskFormat          string   `json:"taskFormat"`
	AllowedContexts     []string `json:"allowedContexts"`
	AllowedProjects     []string `json:"allowedProjects"`
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
	if err := validateFileConfigKeys(raw); err != nil {
		return err
	}

	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	return decoder.Decode(dst)
}

func validateFileConfigKeys(raw any) error {
	root, ok := raw.(map[string]any)
	if !ok {
		return nil
	}
	if err := validateJSONObjectKeys(root, map[string]struct{}{
		"dir": {}, "files": {}, "behaviour": {}, "colors": {},
	}); err != nil {
		return err
	}
	if files, ok := root["files"].(map[string]any); ok {
		if err := validateJSONObjectKeys(files, map[string]struct{}{
			"todo": {}, "done": {}, "report": {},
		}); err != nil {
			return err
		}
	}
	if behaviour, ok := root["behaviour"].(map[string]any); ok {
		if err := validateJSONObjectKeys(behaviour, map[string]struct{}{
			"force": {}, "preserveLineNumbers": {}, "autoArchive": {}, "enableUUID": {},
			"priorityOnAdd": {}, "verbose": {}, "defaultAction": {}, "sourceVar": {},
			"sentenceDelimiters": {}, "taskFormat": {}, "allowedContexts": {}, "allowedProjects": {},
		}); err != nil {
			return err
		}
	}
	if colors, ok := root["colors"].(map[string]any); ok {
		if err := validateJSONObjectKeys(colors, map[string]struct{}{
			"priA": {}, "priB": {}, "priC": {}, "priD": {}, "priE": {}, "priF": {},
			"priG": {}, "priH": {}, "priI": {}, "priJ": {}, "priK": {}, "priL": {},
			"priM": {}, "priN": {}, "priO": {}, "priP": {}, "priQ": {}, "priR": {},
			"priS": {}, "priT": {}, "priU": {}, "priV": {}, "priW": {}, "priX": {},
			"priY": {}, "priZ": {}, "colorDone": {}, "colorProject": {}, "colorContext": {},
			"colorDate": {}, "colorNumber": {}, "colorMeta": {}, "map": {},
		}); err != nil {
			return err
		}
	}
	return nil
}

func validateJSONObjectKeys(object map[string]any, allowed map[string]struct{}) error {
	for key := range object {
		if _, ok := allowed[key]; !ok {
			return fmt.Errorf("unknown field %q", key)
		}
	}
	return nil
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
