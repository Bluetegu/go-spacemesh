package hare

import (
	"bytes"
	"encoding/binary"
	"github.com/go-ethereum/common/math"
	"hash/fnv"
)

const (
	Passive = 0
	Active = 1
	Leader = 2
)

type Rolacle interface {
	Role(rq RoleRequest) RoleSignature
	ValidateRole(rq RoleRequest, sig RoleSignature) bool
}

type RoleRequest struct {
	pubKey PubKey
	layerId LayerId
	k uint32
}

func (roleRequest *RoleRequest) bytes() []byte {
	var binBuf bytes.Buffer
	binary.Write(&binBuf, binary.BigEndian, roleRequest)

	return binBuf.Bytes()
}

type MockOracle struct {
	roles map[uint32]uint8
	isLeaderTaken bool
}

func (mockOracle *MockOracle) NewMockOracle() {
	mockOracle.roles = make(map[uint32]uint8)
	mockOracle.isLeaderTaken = false
}

func (roleRequest *RoleRequest) Id() uint32 {
	h := fnv.New32()
	h.Write(roleRequest.bytes())
	return h.Sum32()
}

func (mockOracle *MockOracle) Role(rq RoleRequest) RoleSignature {
	i := rq.Id()

	if !mockOracle.isLeaderTaken {
		mockOracle.roles[i] = Leader
		mockOracle.isLeaderTaken = true
		return RoleSignature{}
	}

	// check if exist
	if _, exist := mockOracle.roles[i]; exist {
		return RoleSignature{}
	}

	if i < math.MaxUint32 / 2 {
		mockOracle.roles[i] = Active
	} else {
		mockOracle.roles[i] = Passive
	}

	return RoleSignature{}
}

func (mockOracle *MockOracle) ValidateRole(role byte, rq RoleRequest, sig RoleSignature) bool {
	return mockOracle.roles[rq.Id()] == role && sig == RoleSignature{}
}
