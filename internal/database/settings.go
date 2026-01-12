package database

import "database/sql"

func GetSetting(key string) (string, bool) {
	d, err := getDefaultDB()
	if err != nil {
		return "", false
	}
	return d.GetSetting(key)
}

func (d *Database) GetSetting(key string) (string, bool) {
	var value sql.NullString
	err := d.DB.QueryRow("SELECT value FROM settings WHERE key = ?", key).Scan(&value)
	if err != nil {
		return "", false
	}
	if value.Valid {
		return value.String, true
	}
	return "", false
}

func SetSetting(key, value string) error {
	d, err := getDefaultDB()
	if err != nil {
		return err
	}
	return d.SetSetting(key, value)
}

func (d *Database) SetSetting(key, value string) error {
	_, err := d.DB.Exec("INSERT INTO settings (key, value) VALUES (?, ?) ON CONFLICT(key) DO UPDATE SET value = excluded.value", key, value)
	return err
}
