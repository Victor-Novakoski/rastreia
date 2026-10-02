package delivery

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"

	"github.com/jackc/pgx/v5"

	"github.com/Victor-Novakoski/rastreia/internal/apperr"
	"github.com/Victor-Novakoski/rastreia/internal/auth"
	"github.com/Victor-Novakoski/rastreia/internal/store"
)

const maxIdempotencyKey = 255

// CreateIdempotent is Create for clients that may retry: the same key from
// the same user within 24 hours returns the delivery created the first time
// (replayed is true) instead of creating another one. Reusing a key with a
// different request is rejected.
func (s *Service) CreateIdempotent(ctx context.Context, actor auth.Claims, key string, in CreateInput) (d Delivery, replayed bool, err error) {
	v := apperr.Validator{}
	v.Check(validIdempotencyKey(key), "idempotency_key", "must have 1 to 255 visible ASCII characters")
	if err := v.Err(); err != nil {
		return Delivery{}, false, err
	}
	if err := s.validateCreate(ctx, actor.CarrierID, &in); err != nil {
		return Delivery{}, false, err
	}
	hash, err := requestHash(in)
	if err != nil {
		return Delivery{}, false, err
	}

	actorID := actor.UserID
	err = s.retryOnCodeCollision(ctx, func(q Store) error {
		k := store.GetIdempotencyKeyParams{UserID: actorID, Key: key}
		if err := q.DeleteExpiredIdempotencyKey(ctx, store.DeleteExpiredIdempotencyKeyParams(k)); err != nil {
			return err
		}
		_, err := q.ReserveIdempotencyKey(ctx, store.ReserveIdempotencyKeyParams{UserID: actorID, Key: key, RequestHash: hash})
		if errors.Is(err, pgx.ErrNoRows) {
			d, err = s.replay(ctx, q, k, hash)
			replayed = err == nil
			return err
		}
		if err != nil {
			return err
		}
		if d, err = s.insert(ctx, q, actor, in); err != nil {
			return err
		}
		return q.SetIdempotencyKeyDelivery(ctx, store.SetIdempotencyKeyDeliveryParams{UserID: actorID, Key: key, DeliveryID: &d.ID})
	})
	if err == nil && !replayed {
		s.announce(ctx, d.CarrierID, d.ID, d.TrackingCode, d.Status, false)
	}
	return d, replayed, err
}

func (s *Service) replay(ctx context.Context, q Store, k store.GetIdempotencyKeyParams, hash string) (Delivery, error) {
	saved, err := q.GetIdempotencyKey(ctx, k)
	if err != nil {
		return Delivery{}, err
	}
	if saved.RequestHash != hash {
		return Delivery{}, &apperr.ValidationError{Fields: map[string]string{
			"idempotency_key": "was already used with a different request",
		}}
	}
	if saved.DeliveryID == nil {
		// The delivery the key pointed to was deleted.
		return Delivery{}, apperr.ErrNotFound
	}
	d, err := q.GetDelivery(ctx, *saved.DeliveryID)
	if err != nil {
		return Delivery{}, notFound(err)
	}
	return fromStore(d), nil
}

// requestHash fingerprints the normalized input, so the same request sent
// with different spacing or casing still matches.
func requestHash(in CreateInput) (string, error) {
	b, err := json.Marshal(in)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:]), nil
}

func validIdempotencyKey(key string) bool {
	if key == "" || len(key) > maxIdempotencyKey {
		return false
	}
	for _, c := range []byte(key) {
		if c < 0x21 || c > 0x7e {
			return false
		}
	}
	return true
}
