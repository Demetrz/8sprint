package main

import (
	"database/sql"
	"math/rand"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	_ "modernc.org/sqlite"
)

var (
	// randSource источник псевдо случайных чисел.
	// Для повышения уникальности в качестве seed
	// используется текущее время в unix формате (в виде числа)
	randSource = rand.NewSource(time.Now().UnixNano())
	// randRange использует randSource для генерации случайных чисел
	randRange = rand.New(randSource)
)

// getTestParcel возвращает тестовую посылку
func getTestParcel() Parcel {
	return Parcel{
		Client:    1000,
		Status:    ParcelStatusRegistered,
		Address:   "test",
		CreatedAt: time.Now().UTC().Format(time.RFC3339),
	}
}

// getTestDB возвращает подключение к существующей БД
func getTestDB(t *testing.T) *sql.DB {
	db, err := sql.Open("sqlite", "tracker.db")
	require.NoError(t, err)
	return db
}

// TestAddGetDelete проверяет добавление, получение и удаление посылки
func TestAddGetDelete(t *testing.T) {
	// prepare
	db := getTestDB(t)
	defer db.Close()

	store := NewParcelStore(db)
	parcel := getTestParcel()

	// add
	id, err := store.Add(parcel)
	require.NoError(t, err)
	require.Greater(t, id, 0)

	// get
	retrieved, err := store.Get(id)
	require.NoError(t, err)

	// проверьте, что значения всех полей в полученном объекте совпадают со значениями полей в переменной parcel
	require.Equal(t, id, retrieved.Number)
	require.Equal(t, parcel.Client, retrieved.Client)
	require.Equal(t, parcel.Status, retrieved.Status)
	require.Equal(t, parcel.Address, retrieved.Address)
	require.Equal(t, parcel.CreatedAt, retrieved.CreatedAt)

	// delete
	err = store.Delete(id)
	require.NoError(t, err)

	// проверьте, что посылку больше нельзя получить из БД
	_, err = store.Get(id)
	require.Error(t, err)
	require.Contains(t, err.Error(), "посылка не найдена")
}

// TestSetAddress проверяет обновление адреса
func TestSetAddress(t *testing.T) {
	// prepare
	db := getTestDB(t)
	defer db.Close()

	store := NewParcelStore(db)

	// add
	parcel := getTestParcel()
	id, err := store.Add(parcel)
	require.NoError(t, err)
	require.Greater(t, id, 0)

	// set address
	newAddress := "new test address"
	err = store.SetAddress(id, newAddress)
	require.NoError(t, err)

	// check
	retrieved, err := store.Get(id)
	require.NoError(t, err)
	require.Equal(t, newAddress, retrieved.Address)

	// Test that address cannot be changed for non-registered parcel
	err = store.SetStatus(id, ParcelStatusSent)
	require.NoError(t, err)

	err = store.SetAddress(id, "another address")
	require.Error(t, err)
	require.Contains(t, err.Error(), "нельзя изменить адрес")
}

// TestSetStatus проверяет обновление статуса
func TestSetStatus(t *testing.T) {
	// prepare
	db := getTestDB(t)
	defer db.Close()

	store := NewParcelStore(db)

	// add
	parcel := getTestParcel()
	id, err := store.Add(parcel)
	require.NoError(t, err)
	require.Greater(t, id, 0)

	// set status
	err = store.SetStatus(id, ParcelStatusSent)
	require.NoError(t, err)

	// check
	retrieved, err := store.Get(id)
	require.NoError(t, err)
	require.Equal(t, ParcelStatusSent, retrieved.Status)
}

// TestGetByClient проверяет получение посылок по идентификатору клиента
func TestGetByClient(t *testing.T) {
	// prepare
	db := getTestDB(t)
	defer db.Close()

	store := NewParcelStore(db)

	parcels := []Parcel{
		getTestParcel(),
		getTestParcel(),
		getTestParcel(),
	}
	parcelMap := map[int]Parcel{}

	// задаём всем посылкам один и тот же идентификатор клиента
	client := randRange.Intn(10_000_000)
	parcels[0].Client = client
	parcels[1].Client = client
	parcels[2].Client = client

	// add
	for i := 0; i < len(parcels); i++ {
		id, err := store.Add(parcels[i])
		require.NoError(t, err)
		require.Greater(t, id, 0)

		parcels[i].Number = id
		parcelMap[id] = parcels[i]
	}

	// get by client
	storedParcels, err := store.GetByClient(client)
	require.NoError(t, err)
	require.Len(t, storedParcels, 3)

	// check
	for _, parcel := range storedParcels {
		expected, exists := parcelMap[parcel.Number]
		require.True(t, exists)

		require.Equal(t, expected.Client, parcel.Client)
		require.Equal(t, expected.Status, parcel.Status)
		require.Equal(t, expected.Address, parcel.Address)
		require.Equal(t, expected.CreatedAt, parcel.CreatedAt)
	}

	// cleanup - удаляем тестовые посылки
	for id := range parcelMap {
		// сначала возвращаем статус в registered для удаления
		store.SetStatus(id, ParcelStatusRegistered)
		store.Delete(id)
	}
}

// TestDeleteNonRegistered проверяет, что нельзя удалить посылку не в статусе registered
func TestDeleteNonRegistered(t *testing.T) {
	db := getTestDB(t)
	defer db.Close()

	store := NewParcelStore(db)

	// add
	parcel := getTestParcel()
	id, err := store.Add(parcel)
	require.NoError(t, err)

	// change status
	err = store.SetStatus(id, ParcelStatusSent)
	require.NoError(t, err)

	// try to delete - should fail
	err = store.Delete(id)
	require.Error(t, err)
	require.Contains(t, err.Error(), "нельзя удалить посылку")

	// cleanup - возвращаем статус и удаляем
	store.SetStatus(id, ParcelStatusRegistered)
	store.Delete(id)
}

// TestSetAddressNonRegistered проверяет, что нельзя изменить адрес у посылки не в статусе registered
func TestSetAddressNonRegistered(t *testing.T) {
	db := getTestDB(t)
	defer db.Close()

	store := NewParcelStore(db)

	// add
	parcel := getTestParcel()
	id, err := store.Add(parcel)
	require.NoError(t, err)

	// change status first
	err = store.SetStatus(id, ParcelStatusSent)
	require.NoError(t, err)

	// try to change address - should fail
	err = store.SetAddress(id, "new address")
	require.Error(t, err)
	require.Contains(t, err.Error(), "нельзя изменить адрес")

	// cleanup - возвращаем статус и удаляем
	store.SetStatus(id, ParcelStatusRegistered)
	store.Delete(id)
}

// TestGetNonExistent проверяет получение несуществующей посылки
func TestGetNonExistent(t *testing.T) {
	db := getTestDB(t)
	defer db.Close()

	store := NewParcelStore(db)

	_, err := store.Get(999999)
	require.Error(t, err)
	require.Contains(t, err.Error(), "посылка не найдена")
}

// TestGetByClientNonExistent проверяет получение посылок для несуществующего клиента
func TestGetByClientNonExistent(t *testing.T) {
	db := getTestDB(t)
	defer db.Close()

	store := NewParcelStore(db)

	// используем очень большое число, которое вряд ли существует
	parcels, err := store.GetByClient(999999999)
	require.NoError(t, err)
	require.Empty(t, parcels)
}
