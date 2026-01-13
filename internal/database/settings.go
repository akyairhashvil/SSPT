package database

import "context"

func (d *Database) GetSetting(ctx context.Context, key string) (string, bool) {
	type settingResult struct {
		value string
		ok    bool
	}
	result, err := withDBContextResult(d, ctx, func(ctx context.Context) (settingResult, error) {
		var value *string
		err := d.DB.QueryRowContext(ctx, "SELECT value FROM settings WHERE key = ?", key).Scan(&value)
		if err != nil {
			return settingResult{}, wrapErr(EntitySetting, OpGet, 0, err)
		}
		if value != nil {
			return settingResult{value: *value, ok: true}, nil
		}
		return settingResult{}, nil
	})
	if err != nil {
		return "", false
	}
	return result.value, result.ok
}

func (d *Database) SetSetting(ctx context.Context, key, value string) error {
	return d.withDBContext(ctx, func(ctx context.Context) error {
		_, err := d.DB.ExecContext(ctx, "INSERT INTO settings (key, value) VALUES (?, ?) ON CONFLICT(key) DO UPDATE SET value = excluded.value", key, value)
		return wrapErr(EntitySetting, OpUpdate, 0, err)
	})
}
