package attributes

import "fmt"

type DATA struct {
	Content []byte
	Header  *AttributeHeader
}

func (data *DATA) SetHeader(header *AttributeHeader) {
	data.Header = header
}

func (data DATA) GetHeader() AttributeHeader {
	return *data.Header
}

func (data DATA) FindType() string {
	return data.Header.GetType()
}
func (data DATA) IsNoNResident() bool {
	return data.Header.IsNoNResident()
}

func (data DATA) ShowInfo() {
	fmt.Printf("type %s %t \n", data.FindType(), data.IsNoNResident())
}
