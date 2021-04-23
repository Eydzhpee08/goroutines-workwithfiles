package wallet

import (
	"fmt"
	"github.com/anonimous-arn/wallet/pkg/types"
	"github.com/google/uuid"
	"io/ioutil"
	"os"
	"reflect"
	"testing"
)

type testService struct {
	*Service
}

type testAccount struct {
	phone		types.Phone
	balance		types.Money
	payments	[]struct{
		amount		types.Money
		category	types.PaymentCategory
	}
}

var defaultTestAccount = testAccount{
	phone:		"+992000000001",
	balance: 	10_000_00,
	payments: 	[]struct{
		amount		types.Money
		category	types.PaymentCategory
	}{
		{amount: 1_000_00, category: "auto"},
	},
}

func newTestService() *testService {
	return &testService{Service: &Service{}}
}

func (s *testService) addAccount(data testAccount) (*types.Account, []*types.Payment, error) {
	account, err := s.RegisterAccount(data.phone)
	if err != nil {
		return nil, nil, fmt.Errorf("can't regist account,  error = %v", err)
	}

	err = s.Deposit(account.ID, data.balance)
	if err != nil {
		return nil, nil, fmt.Errorf("can't deposity account, error = %v", err)
	}

	payments := make([]*types.Payment, len(data.payments))
	for i, payment := range data.payments {
		payments[i], err = s.Pay(account.ID, payment.amount, payment.category)
		if err != nil {
			return nil, nil, fmt.Errorf("can't make payment, error = %v", err)
		}
	}

	return account, payments, nil
}

func TestService_FindAccountByID_success(t *testing.T) {
	svc := &Service{}

	account, _ := svc.RegisterAccount("+992000000001")

	acc, e := svc.FindAccountByID(account.ID)

	if e != nil {
		t.Error(e)
	}

	if !reflect.DeepEqual(account, acc) {
		t.Error("Accounts doesn't match")
	}
}

func TestService_FindAccountByID_notFound(t *testing.T) {
	svc := &Service{}

	_, e := svc.FindAccountByID(123)

	if e != ErrAccountNotFound {
		t.Error(e)
	}
}

func TestService_FindPaymentByID_success(t *testing.T) {
	s := newTestService()

	_, payments, err := s.addAccount(defaultTestAccount)
	if err != nil {
		t.Error(err)
		return
	}

	payment := payments[0]
	got, err := s.FindPaymentByID(payment.ID)
	if err != nil {
		t.Errorf("FindPaymentByID(): error = %v", err)
		return
	}

	if !reflect.DeepEqual(payment, got) {
		t.Errorf("FindPaymentByID(): wrong paymen returned = %v", err)
		return
	}
}

func TestService_FindPaymentByID_fail(t *testing.T) {
	s := newTestService()
	_, _, err := s.addAccount(defaultTestAccount)
	if err != nil {
		t.Error(err)
		return
	}

	_, err = s.FindPaymentByID(uuid.New().String())
	if err == nil {
		t.Error("FindPaymentByID(): must return error, returned nil")
		return
	}

	if err != ErrPaymentNotFound {
		t.Errorf("FindPaymentByID(): must retunrn ErrPaymentNotFound, returned = %v", err)
		return
	}
}

func TestService_Reject_success(t *testing.T) {
	s := newTestService()
	_, payments, err := s.addAccount(defaultTestAccount)
	if err != nil {
		t.Error(err)
		return
	}

	payment := payments[0]
	err = s.Reject(payment.ID)
	if err != nil {
		t.Errorf("Reject(): error = %v", err)
		return
	}

	savedPayment, err := s.FindPaymentByID(payment.ID)
	if err != nil {
		t.Errorf("Reject(): can't find payment by id, error = %v", err)
		return
	}
	if savedPayment.Status != types.PaymentStatusFail {
		t.Errorf("Reject(): status can't changed, error = %v", savedPayment)
		return
	}

	savedAccount, err := s.FindAccountByID(payment.AccountID)
	if err != nil {
		t.Errorf("Reject(): can't find account by id, error = %v", err)
		return
	}
	if savedAccount.Balance != defaultTestAccount.balance {
		t.Errorf("Reject(), balance didn't cahnged, account = %v", savedAccount)
	}
}

func TestService_Reject_fail(t *testing.T) {
	s := newTestService()
	_, _, err := s.addAccount(defaultTestAccount)
	if err != nil {
		t.Error(err)
		return
	}

	err = s.Reject(uuid.New().String())
	if err == nil {
		t.Error("Reject(): must be error, returned nil")
		return
	}
}

func TestService_Repeat_success(t *testing.T) {
	s := newTestService()
	_, payments, err := s.addAccount(defaultTestAccount)
	if err != nil {
		t.Error(err)
		return
	}

	payment := payments[0]
	newPayment, nil := s.Repeat(payment.ID)
	if err != nil {
		t.Error(err)
		return
	}

	if payment.ID == newPayment.ID {
		t.Error("repeated payment id not different")
		return
	}

	if payment.AccountID != newPayment.AccountID ||
		payment.Status != newPayment.Status ||
		payment.Category != newPayment.Category ||
		payment.Amount != newPayment.Amount {
		t.Error("some field is not equal the original")
	}
}

func TestService_FavoritePayment_success(t *testing.T) {
	s := newTestService()
	_, payments, err := s.addAccount(defaultTestAccount)
	if err != nil {
		t.Error(err)
		return
	}

	payment := payments[0]

	_, err = s.FavoritePayment(payment.ID, "osh")
	if err != nil {
		t.Error(err)
	}
}

func TestService_FavoritePayment_fail(t *testing.T) {
	s := newTestService()

	_, err := s.FavoritePayment(uuid.New().String(), "osh")
	if err == nil {
		t.Error("FavoritePayment(): must return error, now nil")
	}
}

func TestService_PayFromFavorite_success(t *testing.T) {
	s := newTestService()

	_, payments, err := s.addAccount(defaultTestAccount)
	if err != nil {
		t.Error("PayFromFavorite(): can't get payments")
		return
	}

	payment := payments[0]

	favorite, err := s.FavoritePayment(payment.ID, "osh")
	if err != nil {
		t.Error("PayFromFavorite(): can't add payment to favorite")
		return
	}

	_, err = s.PayFromFavorite(favorite.ID)
	if err != nil {
		t.Error("PayFromFavorite(): can't not pay from favorite")
		return
	}
}

func TestService_PayFromFavorite_fail(t *testing.T) {
	s := newTestService()

	_, err := s.PayFromFavorite(uuid.New().String())
	if err == nil {
		t.Error("PayFromFavorite(): must be error, now returned nil")
	}
}
func TestService_ExportToFile_EmptyData(t *testing.T) {
	svc := &Service{}

	err := svc.ExportToFile("1.txt")
	if err != nil {
		t.Error(err)
	}
	file, err := os.Open("1.txt")
	if err != nil {
		t.Error(err)
	}

	stats, err := file.Stat()
	if err != nil {
		t.Error(err)
	}

	if stats.Size() != 0 {
		t.Error("file must be zero")
	}
}

func TestService_ExportToFile(t *testing.T) {
	svc := &Service{}

	_, err := svc.RegisterAccount("+992000000000")
	if err != nil {
		t.Error(err)
	}

	err = svc.ExportToFile("1.txt")
	if err != nil {
		t.Error(err)
	}
	file, err := os.Open("1.txt")
	if err != nil {
		t.Error(err)
	}

	stats, err := file.Stat()
	if err != nil {
		t.Error(err)
	}

	if stats.Size() == 0 {
		t.Error("file must be zero")
	}
}

func TestService_ImportToFile(t *testing.T) {
	svc := &Service{}

	err := svc.ImportFromFile("1.txt")
	if err != nil {
		t.Error(err)
	}

	k := 0
	for _, account := range svc.accounts {
		if account.Phone == "+992000000000" {
			k++
		}
	}

	if k <= 0 {
		t.Error("incorrect func")
	}
}

func TestSetice_Export(t *testing.T) {
	svc := &Service{}

	account, err := svc.RegisterAccount("+992000000000")
	if err != nil {
		t.Error(err)
	}

	account.Balance = 100

	payment, err := svc.Pay(account.ID, 100, "auto")
	if err != nil {
		t.Error(err)
	}

	_, err = svc.FavoritePayment(payment.ID, "isbraniy")
	if err != nil {
		t.Error(err)
	}

	err = svc.Export(".")
	if err != nil {
		t.Error(err)
	}

	_, err = ioutil.ReadFile("accounts.dump")
	if err != nil {
		t.Error(err)
	}

	_, err = ioutil.ReadFile("payments.dump")
	if err != nil {
		t.Error(err)
	}

	_, err = ioutil.ReadFile("favorites.dump")
	if err != nil {
		t.Error(err)
	}
}

func TestService_Import(t *testing.T) {
	svc := &Service{}

	err := svc.Import(".")
	if err != nil {
		t.Error(err)
	}

	if svc.accounts[0].Phone != "+992000000000" {
		t.Error("incorrect func")
	}
}
func TestService_Import_IfHaveData(t *testing.T) {
	svc := &Service{}

	account, err := svc.RegisterAccount("+992000000000")
	if err != nil {
		t.Error(err)
	}

	account.Balance = 100

	payment, err := svc.Pay(account.ID, 100, "auto")
	if err != nil {
		t.Error(err)
	}

	_, err = svc.FavoritePayment(payment.ID, "isbraniy")
	if err != nil {
		t.Error(err)
	}

	err = svc.Import(".")
	if err != nil {
		t.Error(err)
	}

	if account.Phone == "+992" {
		t.Error("incorrect func")
	}
}

func TestService_HistoryToFile(t *testing.T) {
	svc := &Service{}

	payments := []types.Payment{
		{
			ID:        "1",
			AccountID: 1,
			Amount:    10,
			Category:  "auto",
			Status:    "active",
		},
		{
			ID:        "2",
			AccountID: 1,
			Amount:    10,
			Category:  "auto",
			Status:    "active",
		},
		{
			ID:        "3",
			AccountID: 1,
			Amount:    10,
			Category:  "auto",
			Status:    "active",
		},
		{
			ID:        "4",
			AccountID: 1,
			Amount:    10,
			Category:  "auto",
			Status:    "active",
		},
		{
			ID:        "5",
			AccountID: 1,
			Amount:    10,
			Category:  "auto",
			Status:    "active",
		},
		{
			ID:        "6",
			AccountID: 1,
			Amount:    10,
			Category:  "auto",
			Status:    "active",
		},
		{
			ID:        "7",
			AccountID: 1,
			Amount:    10,
			Category:  "auto",
			Status:    "active",
		},
		{
			ID:        "8",
			AccountID: 1,
			Amount:    10,
			Category:  "auto",
			Status:    "active",
		},
		{
			ID:        "9",
			AccountID: 1,
			Amount:    10,
			Category:  "auto",
			Status:    "active",
		},
		{
			ID:        "10",
			AccountID: 1,
			Amount:    10,
			Category:  "auto",
			Status:    "active",
		},
	}

	err := svc.HistoryToFiles(payments, ".", 3)
	if err != nil {
		t.Error(err)
	}

	err = svc.HistoryToFiles(payments, ".", 4)
	if err != nil {
		t.Error(err)
	}

	err = svc.HistoryToFiles(payments, ".", 5)
	if err != nil {
		t.Error(err)
	}

	err = svc.HistoryToFiles(payments, ".", 10)
	if err != nil {
		t.Error(err)
	}

	err = svc.HistoryToFiles(payments, ".", 11)
	if err != nil {
		t.Error(err)
	}

	err = svc.HistoryToFiles(payments, ".", 1)
	if err != nil {
		t.Error(err)
	}

}

func fileFunc(l int, t *testing.T) {
	files, err := ioutil.ReadDir("./test")
	if err != nil {
		t.Error(err)
	}

	if len(files) != l {
		t.Error("incorrect")
	}

	for _, file := range files {
		err = os.Remove("test/" + file.Name())
		if err != nil {
			t.Error(err)
		}
	}
}

func TestService_SumPayments(t *testing.T) {
	svc := &Service{}

	for i := 0; i < 103; i++ {
		svc.payments = append(svc.payments, &types.Payment{Amount: 1})
	}

	sum := svc.SumPayments(10)
	if sum != 103 {
		t.Error("incoorect")
	}
}

func Benchmark_SumPayments(b *testing.B) {
	svc := &Service{}

	for i := 0; i < 103; i++ {
		svc.payments = append(svc.payments, &types.Payment{Amount: 1})
	}

	result := 103

	for i := 0; i < b.N; i++ {
		sum := svc.SumPayments(result)
		if result != int(sum) {
			b.Fatalf("invalid result, got %v, want %v", sum, result)
		}
	}
}

func Benchmark_FilterPayments(b *testing.B) {
	svc := &Service{}

	account, err := svc.RegisterAccount("+992000000000")
	if err != nil {
		b.Error(err)
	}
	for i := 0; i < 103; i++ {
		svc.payments = append(svc.payments, &types.Payment{AccountID: account.ID, Amount: 1})
	}

	result := 103

	for i := 0; i < b.N; i++ {
		payments, err := svc.FilterPayments(account.ID, result)
		if err != nil {
			b.Error(err)
		}

		if result != len(payments) {
			b.Fatalf("invalid result, got %v, want %v", len(payments), result)
		}
	}
}
