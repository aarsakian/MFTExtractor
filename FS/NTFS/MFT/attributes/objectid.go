package attributes

import (
	"fmt"

	"github.com/aarsakian/MFTExtractor/utils"
)

type ObjectID struct { //unique guID
	ObjID     string //object ID
	OrigVolID string //volume ID
	OrigObjID string //original objID
	OrigDomID string // domain ID
	Header    *AttributeHeader
}

func (objectId *ObjectID) SetHeader(header *AttributeHeader) {
	objectId.Header = header
}

func (objectId ObjectID) GetHeader() AttributeHeader {
	return *objectId.Header
}

func (objectId *ObjectID) Parse(data []byte) {
	utils.Unmarshal(data, objectId)
}

func (objectId ObjectID) FindType() string {
	return objectId.Header.GetType()
}

func (objectId ObjectID) IsNoNResident() bool {
	return objectId.Header.IsNoNResident()
}

func (objectId ObjectID) ShowInfo() {
	fmt.Printf("type %s\n", objectId.FindType())
}
