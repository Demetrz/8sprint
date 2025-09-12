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

// TestAddGetDelete проверяет добавление, получение и удаление посылки
func TestAddGetDelete(t *testing.T) {
	// prepare
	db, err := sql.Open("sqlite", "tracker.db") // настройте подключение к БД
	require.NoError(t, err)
	defer db.Close()

	store := NewParcelStore(db)
	parcel := getTestParcel()

	// add
	// добавьте новую посылку в БД, убедитесь в отсутствии ошибки и наличии идентификатора
	id, err := store.Add(parcel)
	require.NoError(t, err)
	require.Greater(t, id, 0)

	// get
	// получите только что добавленную посылку, убедитесь в отсутствии ошибки
	retrieved, err := store.Get(id)
	require.NoError(t, err)

	// проверьте, что значения всех полей в полученном объекте совпадают со значениями полей в переменной parcel
	require.Equal(t, id, retrieved.Number)
	require.Equal(t, parcel.Client, retrieved.Client)
	require.Equal(t, parcel.Status, retrieved.Status)
	require.Equal(t, parcel.Address, retrieved.Address)
	require.Equal(t, parcel.CreatedAt, retrieved.CreatedAt)

	// delete
	// удалите добавленную посылку, убедитесь в отсутствии ошибки
	err = store.Delete(id)
	require.NoError(t, err)

	// проверьте, что посылку больше нельзя получить из БД
	_, err = store.Get(id)
	require.Error(t, err)
}

// TestSetAddress проверяет обновление адреса
func TestSetAddress(t *testing.T) {
	// prepare
	db, err := sql.Open("sqlite", "tracker.db") // настройте подключение к БД
	require.NoError(t, err)
	defer db.Close()

	store := NewParcelStore(db)

	// add
	// добавьте новую посылку в БД, убедитесь в отсутствии ошибки и наличии идентификатора
	parcel := getTestParcel()
	id, err := store.Add(parcel)
	require.NoError(t, err)
	require.Greater(t, id, 0)

	// set address
	// обновите адрес, убедитесь в отсутствии ошибки
	newAddress := "new test address"
	err = store.SetAddress(id, newAddress)
	require.NoError(t, err)

	// check
	// получите добавленную посылку и убедитесь, что адрес обновился
	retrieved, err := store.Get(id)
	require.NoError(t, err)
	require.Equal(t, newAddress, retrieved.Address)
}

// TestSetStatus проверяет обновление статуса
func TestSetStatus(t *testing.T) {
	// prepare
	db, err := sql.Open("sqlite", "tracker.db") // настройте подключение к БД
	require.NoError(t, err)
	defer db.Close()

	store := NewParcelStore(db)

	// add
	// добавьте новую посылку в БД, убедитесь в отсутствии ошибки и наличии идентификатора
	parcel := getTestParcel()
	id, err := store.Add(parcel)
	require.NoError(t, err)
	require.Greater(t, id, 0)

	// set status
	// обновите статус, убедитесь в отсутствии ошибки
	err = store.SetStatus(id, ParcelStatusSent)
	require.NoError(t, err)

	// check
	// получите добавленную посылку и убедитесь, что статус обновился
	retrieved, err := store.Get(id)
	require.NoError(t, err)
	require.Equal(t, ParcelStatusSent, retrieved.Status)
}

// TestGetByClient проверяет получение посылок по идентификатору клиента
func TestGetByClient(t *testing.T) {
	// prepare
	db, err := sql.Open("sqlite", "tracker.db") // настройте подключение к БД
	require.NoError(t, err)
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
		id, err := store.Add(parcels[i]) // добавьте новую посылку в БД, убедитесь в отсутствии ошибки и наличии идентификатора
		require.NoError(t, err)
		require.Greater(t, id, 0)

		// обновляем идентификатор добавленной у посылки
		parcels[i].Number = id

		// сохраняем добавленную посылку в структуру map, чтобы её можно было легко достать по идентификатору посылки
		parcelMap[id] = parcels[i]
	}

	// get by client
	storedParcels, err := store.GetByClient(client) // получите список посылок по идентификатору клиента, сохранённого в переменной client
	require.NoError(t, err)                         // убедитесь в отсутствии ошибки
	require.Len(t, storedParcels, 3)                // убедитесь, что количество полученных посылок совпадает с количеством добавленных

	// check
	for _, parcel := range storedParcels {
		// в parcelMap лежат добавленные посылки, ключ - идентификатор посылки, значение - сама посылка
		// убедитесь, что все посылки из storedParcels есть в parcelMap
		expected, exists := parcelMap[parcel.Number]
		require.True(t, exists)

		// убедитесь, что значения полей полученных посылок заполнены верно
		require.Equal(t, expected.Client, parcel.Client)
		require.Equal(t, expected.Status, parcel.Status)
		require.Equal(t, expected.Address, parcel.Address)
		require.Equal(t, expected.CreatedAt, parcel.CreatedAt)
	}
}
