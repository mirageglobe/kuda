package ui

// Server represents a MUD server entry.
type Server struct {
	Name    string
	Address string
	Desc    string
}

// GetServers returns the default list of MUD servers.
func GetServers() []Server {
	return []Server{
		{Name: "Aardwolf", Address: "aardmud.org:23", Desc: "popular fantasy mud"},
		{Name: "TorilMUD", Address: "torilmud.com:9999", Desc: "forgotten realms mud"},
		{Name: "Genesis MUD", Address: "genesismud.org:3011", Desc: "the original lpmud"},
		{Name: "Local Echo (dev)", Address: MockAddress, Desc: "in-process echo for testing"},
	}
}
