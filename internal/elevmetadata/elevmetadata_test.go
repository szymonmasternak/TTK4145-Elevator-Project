package elevmetadata

import "testing"

func TestString(t *testing.T) {
	metadata := ElevMetaData{
		SoftwareVersion: "smj2acjkvv4h1zkwjz2ocsn2lkfrjmzf9qn4i2m3",
		IpAddress:       "0.0.0.0",
		PortNumber:      9999,
		Identifier:      "uwvvblrtct",
	}

	jsonString := "{\"software_version\":\"smj2acjkvv4h1zkwjz2ocsn2lkfrjmzf9qn4i2m3\",\"ip_address\":\"0.0.0.0\",\"port_number\":9999,\"identifier\":\"uwvvblrtct\"}"

	if metadata.String() != jsonString {
		t.Errorf("String() = %s, expected %s", metadata.String(), jsonString)
	}
}

func TestGetIPAddressPort(t *testing.T) {
	metadata := ElevMetaData{
		SoftwareVersion: "smj2acjkvv4h1zkwjz2ocsn2lkfrjmzf9qn4i2m3",
		IpAddress:       "0.0.0.0",
		PortNumber:      9999,
		Identifier:      "uwvvblrtct",
	}

	if metadata.GetIPAddressPort() != "0.0.0.0:9999" {
		t.Errorf("GetIPAddressPort() = %s, expected 0.0.0.0:9999", metadata.GetIPAddressPort())
	}
}
