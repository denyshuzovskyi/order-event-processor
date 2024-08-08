package handler

import (
	"testing"
)

func TestPaymentSystemEventHandler_ProcessEvent(t *testing.T) {
	//logger := slog.New(handler.NewNoOpHandler())
	//validate := validator.New()
	//paymentSystemEventHandler := NewPaymentSystemEventHandler(logger, validate)
	//
	//eventJson := `
	//	{
	//		"event_id": "483ec8f8-4864-427b-a878-ca026fd38f88",
	//		"order_id": "97a96c29-7631-4cbc-9559-f8866fb03392",
	//		"user_id": "2c127d70-3b9b-4743-9c2e-74b9f617029f",
	//		"order_status": "cool_order_created",
	//		"updated_at":"2019-01-01T00:00:00Z",
	//		"created_at": "2019-01-01T00:00:00Z"
	//	}
	//`
	//
	//req, err := http.NewRequest("POST", "", bytes.NewBufferString(eventJson))
	//if err != nil {
	//	t.Fatal(err)
	//}
	//req.Header.Set("Content-Type", "application/json")
	//
	//rr := httptest.NewRecorder()
	//
	//paymentSystemEventHandler.Handle(rr, req)
	//
	//if status := rr.Code; status != http.StatusOK {
	//	t.Errorf("handler returned wrong status code: got %v want %v",
	//		status, http.StatusOK)
	//}
}
