package redis

import (
	"context"
	"errors"
	"fmt"
	"time"

	setting "email-send-API/src/Setting"

	goredis "github.com/redis/go-redis/v9"
)

// ClientRedis regroupe les opérations Redis utilisées par l'API.
type ClientRedis struct {
	client *goredis.Client
}

// Ouvrir établit la connexion et empêche le démarrage si Redis est indisponible.
func Ouvrir(cfg *setting.Config) (*ClientRedis, error) {
	if cfg == nil {
		return nil, errors.New("configuration Redis absente")
	}

	client := goredis.NewClient(&goredis.Options{
		Addr:     cfg.AdresseRedis,
		Password: cfg.MotDePasseRedis,
		DB:       cfg.BaseRedis,
	})
	contexte, annuler := context.WithTimeout(context.Background(), 5*time.Second)
	defer annuler()
	if err := client.Ping(contexte).Err(); err != nil {
		_ = client.Close()
		return nil, fmt.Errorf("vérifier la connexion Redis: %w", err)
	}
	return &ClientRedis{client: client}, nil
}

// Fermer libère le pool de connexions Redis.
func (c *ClientRedis) Fermer() error {
	if c == nil || c.client == nil {
		return nil
	}
	return c.client.Close()
}

// VerifierDisponibilite vérifie que Redis répond.
func (c *ClientRedis) VerifierDisponibilite(ctx context.Context) error {
	if c == nil || c.client == nil {
		return errors.New("client Redis indisponible")
	}
	return c.client.Ping(ctx).Err()
}

// Autoriser incrémente atomiquement le compteur et renvoie le quota restant.
func (c *ClientRedis) Autoriser(ctx context.Context, cle string, limite int, fenetre time.Duration) (bool, int64, time.Duration, error) {
	const script = `
	local nombre = redis.call("INCR", KEYS[1])
	if nombre == 1 then
  redis.call("PEXPIRE", KEYS[1], ARGV[1])
end
	local delai = redis.call("PTTL", KEYS[1])
	if delai < 0 then
  redis.call("PEXPIRE", KEYS[1], ARGV[1])
	  delai = tonumber(ARGV[1])
end
	return {nombre, delai}
`
	if c == nil || c.client == nil {
		return false, 0, 0, errors.New("client Redis indisponible")
	}
	resultat, err := c.client.Eval(ctx, script, []string{cle}, fenetre.Milliseconds()).Slice()
	if err != nil {
		return false, 0, 0, fmt.Errorf("évaluer la limite de débit Redis: %w", err)
	}
	if len(resultat) != 2 {
		return false, 0, 0, errors.New("réponse Redis invalide pour la limite de débit")
	}
	nombre, nombreValide := resultat[0].(int64)
	delaiMillisecondes, delaiValide := resultat[1].(int64)
	if !nombreValide || !delaiValide || delaiMillisecondes < 0 {
		return false, 0, 0, errors.New("valeurs Redis invalides pour la limite de débit")
	}

	restante := int64(limite) - nombre
	if restante < 0 {
		restante = 0
	}
	return nombre <= int64(limite), restante, time.Duration(delaiMillisecondes) * time.Millisecond, nil
}
