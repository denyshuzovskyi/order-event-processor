package postgresql

import (
	"context"
	"fmt"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"hash/fnv"
	"order-event-processor/internal/model"
)

type Storage struct {
	pool *pgxpool.Pool
}

func New(pool *pgxpool.Pool) *Storage {
	return &Storage{
		pool: pool,
	}
}

func (s *Storage) RunInTransaction(run func()) error {
	tx, err := s.pool.Begin(context.Background())
	if err != nil {
		return err
	}
	run()
	if err := tx.Commit(context.Background()); err != nil {
		return tx.Rollback(context.Background())
	}
	return nil
}

func (s *Storage) AcquireLock(id string) error {
	const op = "storage.postgresql.SaveEvent"
	h := fnv.New32a()
	h.Write([]byte(id))
	sum32 := h.Sum32()

	query := `SELECT pg_advisory_xact_lock($1)`
	_, err := s.pool.Exec(context.Background(), query, sum32)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}

func (s *Storage) SaveOrderEvent(orderEvent model.OrderEvent) error {
	const op = "storage.postgresql.SaveEvent"

	query := `INSERT INTO order_events (event_id, order_id, user_id, order_status, is_final, updated_at, created_at) VALUES ($1, $2, $3, $4, $5, $6, $7)`
	_, err := s.pool.Exec(context.Background(), query, orderEvent.EventID, orderEvent.OrderID, orderEvent.UserID, orderEvent.OrderStatus, orderEvent.IsFinal, orderEvent.UpdatedAt, orderEvent.CreatedAt)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}

func (s *Storage) UpdateOrderEventFinalStatus(eventId string) error {
	const op = "storage.postgresql.UpdateOrderEventFinalStatus"

	query := `UPDATE order_events SET is_final = TRUE WHERE event_id = $1;`
	_, err := s.pool.Exec(context.Background(), query, eventId)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}

func (s *Storage) ExistsOrderEventWithEventId(eventId string) (bool, error) {
	const op = "storage.postgresql.ExistsOrderEventWithEventId"
	query := `SELECT EXISTS(SELECT 1 FROM order_events WHERE event_id = $1)`
	row := s.pool.QueryRow(context.Background(), query, eventId)

	var exists bool
	err := row.Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("%s: scan row: %w", op, err)
	}

	return exists, nil
}

func (s *Storage) ExistsOrderEventForOrderIdFinalAndInOrder(orderId string) (bool, error) {
	const op = "storage.postgresql.ExistsOrderEventForOrderIdWithIsFinalStatus"
	query := `SELECT EXISTS(SELECT 1 FROM order_events WHERE order_id = $1 AND is_final = TRUE AND is_in_order = TRUE)`
	row := s.pool.QueryRow(context.Background(), query, orderId)

	var exists bool
	err := row.Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("%s: scan row: %w", op, err)
	}

	return exists, nil
}

func (s *Storage) ExistsOrderEventForOrderIdWithStatus(orderId string, orderStatus model.OrderStatus) (bool, error) {
	const op = "storage.postgresql.ExistsOrderEventForOrderIdWithStatus"
	query := `SELECT EXISTS(SELECT 1 FROM order_events WHERE order_id = $1 AND order_status = $2)`
	row := s.pool.QueryRow(context.Background(), query, orderId, orderStatus)

	var exists bool
	err := row.Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("%s: scan row: %w", op, err)
	}

	return exists, nil
}

func (s *Storage) InsertOrUpdateOrder(order model.Order) error {
	const op = "storage.postgresql.InsertOrUpdateOrder"

	query := `INSERT INTO orders (order_id, user_id, order_status, is_final, updated_at, created_at) VALUES ($1, $2, $3, $4, $5, $6) ON CONFLICT (order_id) 
				  DO UPDATE SET order_status = EXCLUDED.order_status, is_final = EXCLUDED.is_final, updated_at = EXCLUDED.updated_at, created_at = EXCLUDED.created_at`
	_, err := s.pool.Exec(context.Background(), query, order.OrderID, order.UserID, order.OrderStatus, order.IsFinal, order.UpdatedAt, order.CreatedAt)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}

func (s *Storage) InsertOrderEventsOrUpdateIsInOrder(orderEvents ...model.OrderEvent) error {
	const op = "storage.postgresql.InsertOrderEventsOrUpdateIsInOrder"

	batch := &pgx.Batch{}

	for _, orderEvent := range orderEvents {
		query := `INSERT INTO order_events (event_id, order_id, user_id, order_status, is_final, updated_at, created_at, is_in_order) 
				  VALUES ($1, $2, $3, $4, $5, $6, $7, $8) 
				  ON CONFLICT (event_id) 
				  DO UPDATE SET is_in_order = EXCLUDED.is_in_order`
		batch.Queue(query, orderEvent.EventID, orderEvent.OrderID, orderEvent.UserID, orderEvent.OrderStatus, orderEvent.IsFinal, orderEvent.UpdatedAt, orderEvent.CreatedAt, orderEvent.InOrder)
	}

	batchResults := s.pool.SendBatch(context.Background(), batch)
	defer batchResults.Close()

	for i := 0; i < len(orderEvents); i++ {
		_, err := batchResults.Exec()
		if err != nil {
			return fmt.Errorf("%s: %w", op, err)
		}
	}

	return nil
}

func (s *Storage) GetAllEventsByOrderId(orderId string) ([]model.OrderEvent, error) {
	const op = "storage.postgresql.GetAllOrders"
	query := `SELECT event_id, order_id, user_id, order_status, is_final, updated_at, created_at, is_in_order FROM order_events WHERE order_id = $1`

	rows, err := s.pool.Query(context.Background(), query, orderId)
	if err != nil {
		return nil, fmt.Errorf("%s: query: %w", op, err)
	}
	defer rows.Close()

	var orderEvents []model.OrderEvent
	for rows.Next() {
		var orderEvent model.OrderEvent
		err := rows.Scan(&orderEvent.EventID, &orderEvent.OrderID, &orderEvent.UserID, &orderEvent.OrderStatus, &orderEvent.IsFinal, &orderEvent.UpdatedAt, &orderEvent.CreatedAt, &orderEvent.InOrder)
		if err != nil {
			return nil, fmt.Errorf("%s: scan row: %w", op, err)
		}
		orderEvents = append(orderEvents, orderEvent)
	}

	if rows.Err() != nil {
		return nil, fmt.Errorf("%s: iterating over rows: %w", op, rows.Err())
	}

	return orderEvents, nil
}

func (s *Storage) GetAllOrders() ([]model.Order, error) {
	const op = "storage.postgresql.GetAllOrders"
	query := `SELECT order_id, user_id, order_status, is_final, updated_at, created_at FROM orders`

	rows, err := s.pool.Query(context.Background(), query)
	if err != nil {
		return nil, fmt.Errorf("%s: query: %w", op, err)
	}
	defer rows.Close()

	var orders []model.Order
	for rows.Next() {
		var order model.Order
		err := rows.Scan(&order.OrderID, &order.UserID, &order.OrderStatus, &order.IsFinal, &order.CreatedAt, &order.UpdatedAt)
		if err != nil {
			return nil, fmt.Errorf("%s: scan row: %w", op, err)
		}
		orders = append(orders, order)
	}

	if rows.Err() != nil {
		return nil, fmt.Errorf("%s: iterating over rows: %w", op, rows.Err())
	}

	return orders, nil
}

func (s *Storage) DeleteAllFromOrderEvents() error {
	const op = "storage.postgresql.DeleteAllFromOrderEvents"

	query := `DELETE FROM order_events;`
	_, err := s.pool.Exec(context.Background(), query)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}

func (s *Storage) DeleteAllFromOrders() error {
	const op = "storage.postgresql.DeleteAllFromOrders"

	query := `DELETE FROM orders;`
	_, err := s.pool.Exec(context.Background(), query)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}
