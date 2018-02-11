package netmanager

import (
	"sync"

	"github.com/Scalingo/sand/api/types"
)

type ManagerMap struct {
	managers map[types.NetworkType]NetManager
	m        *sync.Mutex
}

func NewManagerMap() ManagerMap {
	return ManagerMap{
		managers: make(map[types.NetworkType]NetManager),
		m:        &sync.Mutex{},
	}
}

func (m *ManagerMap) Set(t types.NetworkType, nm NetManager) {
	m.m.Lock()
	defer m.m.Unlock()
	m.managers[t] = nm
}

func (m *ManagerMap) Get(t types.NetworkType) NetManager {
	m.m.Lock()
	defer m.m.Unlock()
	return m.managers[t]
}
