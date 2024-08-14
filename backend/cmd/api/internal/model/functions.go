package model

type Function struct {
	FunctionID            int32  `json:"functionid"`
	Function              string `json:"function"`
	Description           string `json:"description"`
	DataCenterEnvironment string `json:"datacenterenvironment"`
}
