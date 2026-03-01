package domain

// Addressbook maps chainID (as string) -> name -> address.
type Addressbook map[string]map[string]string

// AddressbookEntry represents a single entry in the addressbook.
type AddressbookEntry struct {
	Name    string
	Address string
}
