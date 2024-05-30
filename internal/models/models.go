package models

type File struct {
	FileID int    `json:"fileID" yaml:"fileID"`
	GUID   string `json:"guid" yaml:"guid"`
	Type   int    `json:"type" yaml:"type"`
}

type Meta struct {
	GUID string `json:"guid" yaml:"guid"`
}

type Asset struct {
	MonoBehaviour AssetMonoBehavior `json:"MonoBehaviour" yaml:"MonoBehaviour"`
	Sprite        AssetSprite       `json:"Sprite" yaml:"Sprite"`
}

// Assets/MonoBehaviour/{m_Name}.asset
type AssetMonoBehavior struct {
	MName               string                `json:"m_Name" yaml:"m_Name"`
	ItemName            string                `json:"itemName" yaml:"itemName"`
	ItemCategory        string                `json:"itemCategory" yaml:"itemCategory"`
	ItemSprite          File                  `json:"itemSprite" yaml:"itemSprite"`
	DefaultGiftLevel    int                   `json:"defaultGiftLevel" yaml:"defaultGiftLevel"`
	SellValue           int                   `json:"sellValue" yaml:"sellValue"`
	Store               int                   `json:"store" yaml:"store"`
	ItemForSale         File                  `json:"itemForSale" yaml:"itemForSale"`
	ActiveAtLocation    File                  `json:"activeAtLocation" yaml:"activeAtLocation"`
	CropProductionGuide []CropProductionGuide `json:"cropProductionGuide" yaml:"cropProductionGuide"`
	LootTable           []ProducesItem        `json:"lootTable" yaml:"lootTable"`
}

type CropProductionGuide struct {
	MachineType         int          `json:"machineType" yaml:"machineType"`
	ProduceDuration     int          `json:"produceDuration" yaml:"produceDuration"`
	ProducesItem        ProducesItem `json:"producesItem" yaml:"producesItem"`
	ExtraPickPercent    float64      `json:"extraPickPercent" yaml:"extraPickPercent"`
	PickAmount          int          `json:"pickAmount" yaml:"pickAmount"`
	MaxProductionCycles int          `json:"maxProductionCycles" yaml:"maxProductionCycles"`
	StageSprites        []File       `json:"stageSprites" yaml:"stageSprites"`
}

type ProducesItem struct {
	Loot          int  `json:"loot" yaml:"loot"`
	LootTable     File `json:"lootTable" yaml:"lootTable"`
	ItemToDrop    File `json:"itemToDrop" yaml:"itemToDrop"`
	PercentChance int  `json:"percentChance" yaml:"percentChance"`
}

// Assets/Sprite/{m_Name}.asset
type AssetSprite struct {
	MName string `json:"m_Name" yaml:"m_Name"`
	MRect Rect   `json:"m_Rect" yaml:"m_Rect"`
	MRD   struct {
		Texture File `json:"m_Texture" yaml:"texture"` // Assets/Texture2D/{m_Name}.asset
	} `json:"m_RD" yaml:"m_RD"`
}

type Rect struct {
	X      int `json:"x" yaml:"x"` // Offset from LEFT
	Y      int `json:"y" yaml:"y"` // Offset from BOTTOM
	Width  int `json:"width" yaml:"width"`
	Height int `json:"height" yaml:"height"`
}
