package data

import (
	"context"
	"database/sql"
	"time"
)

type Permissions []string

func (p Permissions) Include(code string) bool {
	for _, permission := range p {
		if permission == code {
			return true
		}
	}

	return false
}

type PermissionModel struct {
	DB *sql.DB
}

func (model PermissionModel) GetAllForUser(userID int64) (Permissions, error) {
	SQL := `SELECT p.code FROM permissions p 
			INNER JOIN users_permissions up ON up.permission_id=p.id
			INNER JOIN users u ON up.user_id=u.id
			WHERE u.id=$1`

	args := []interface{}{userID}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	rows, err := model.DB.QueryContext(ctx, SQL, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	permissions := Permissions{}
	for rows.Next() {
		var code string
		err = rows.Scan(&code)
		if err != nil {
			return nil, err
		}

		permissions = append(permissions, code)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return permissions, nil
}

func (model PermissionModel) AddForUser(userID int64, permissionCode ...string) error {
	SQL := `INSERT INTO users_permissions (user_id, permission_id)
			SELECT $1, permissions.id FROM permissions WHERE permissions.code = ANY($2)`

	args := []interface{}{userID, permissionCode}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	_, err := model.DB.ExecContext(ctx, SQL, args...)
	return err
}
