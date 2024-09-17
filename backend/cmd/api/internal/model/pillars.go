package model

// Pillar represents a record from the pillars table. These are used to organize Questions into like groups.
type Pillar struct {
	PillarID int32  `json:"pillarid"`
	Pillar   string `json:"pillar"`
	Order    int    `json:"order"`
}
