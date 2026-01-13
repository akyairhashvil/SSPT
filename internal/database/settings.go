package database

func (d *Database) GetSetting(key string) (string, bool) {
	var value *string
	err := d.DB.QueryRow("SELECT value FROM settings WHERE key = ?", key).Scan(&value)
	if err != nil {
		return "", false
	}
	if value != nil {
		return *value, true
	}
	return "", false
}

func (d *Database) SetSetting(key, value string) error {
	_, err := d.DB.Exec("INSERT INTO settings (key, value) VALUES (?, ?) ON CONFLICT(key) DO UPDATE SET value = excluded.value", key, value)
	return err
}
