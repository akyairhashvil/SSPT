package database

import "context"

func (d *Database) GetSetting(ctx context.Context, key string) (string, bool) {
	ctx, cancel := d.withTimeout(ctx, defaultDBTimeout)
	defer cancel()
	var value *string
	err := d.DB.QueryRowContext(ctx, "SELECT value FROM settings WHERE key = ?", key).Scan(&value)
	if err != nil {
		return "", false
	}
	if value != nil {
		return *value, true
	}
	return "", false
}

func (d *Database) SetSetting(ctx context.Context, key, value string) error {
	ctx, cancel := d.withTimeout(ctx, defaultDBTimeout)
	defer cancel()
	_, err := d.DB.ExecContext(ctx, "INSERT INTO settings (key, value) VALUES (?, ?) ON CONFLICT(key) DO UPDATE SET value = excluded.value", key, value)
	return err
}
