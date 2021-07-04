package dbrepo

import (
	"context"
	"errors"
	"log"
	"time"

	"github.com/fangjjcs/bookings-app/pkg/models"
	"golang.org/x/crypto/bcrypt"
)
func (m *postgresDBRepo) AllUsers() bool{
	return true
}

func (m *postgresDBRepo) InsertReservations(res models.Reservations) (int, error){
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	var newID int
	stmt := `insert into reservations 
	(first_name, last_name, email, phone, start_date, end_date, room_id, created_at, updated_at)
	 values($1, $2, $3, $4, $5, $6, $7, $8, $9) returning id`
	
    err := m.DB.QueryRowContext(ctx, stmt,
	res.FirstName,
	res.LastName,
	res.Email,
	res.Phone,
	res.StartDate,
	res.EndDate,
	res.RoomID,
	time.Now(),
	time.Now(),
	).Scan(&newID)

	if err != nil{
		return 0, err
	}
	return newID, nil
}

// Insert a room restriction into database
func (m *postgresDBRepo) InsertRoomRestriction(res models.RoomRestrictions) error {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()


	stmt := `insert into room_restrictions 
	(start_date, end_date, room_id, reservation_id,created_at, updated_at, restriction_id)
	 values($1, $2, $3, $4, $5, $6, $7)`
	
	 _, err := m.DB.ExecContext(ctx, stmt,
		res.StartDate,
		res.EndDate,
		res.RoomID,
		res.ReservationID,
		time.Now(),
		time.Now(),
		res.RestrictionID,
		)
	if err!=nil{
		return err
	}

	return nil
}


// Search Availability
func (m *postgresDBRepo) SearchAvailabilityByDatesAndRoomID(start, end time.Time, roomID int) (bool,error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	log.Println(start,end, roomID)
	var count int
	stmt := `select count(id) from room_restrictions where
	        $1 < end_date and $2 > start_date and room_id = $3;`
	
	err := m.DB.QueryRowContext(ctx, stmt , start, end, roomID).Scan(&count)
	if err!=nil{
		return false, err
	}

	log.Println(count)
	// assumes there exists only 1 room
	if count == 0 {
		return true, nil
	}
	return false, nil
}


func (m *postgresDBRepo) SearchAvailibilityForAllRooms(start, end time.Time) ([]models.Room, error){
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	var rooms []models.Room
	query :=`
	select
		r.id, r.room_name
	from
		rooms r 
	where r.id not in 
		(select rr.room_id from room_restrictions rr where rr.start_date <$2 and rr.end_date>$1)
		`
	rows, err := m.DB.QueryContext(ctx,query,start,end)
	if err != nil{
		return nil, err
	}
	var room models.Room
	for rows.Next(){
		err := rows.Scan(&room.ID,&room.RoomName)
		if err!= nil{
			return nil, err
		}
		rooms = append(rooms, room)
	}

	return rooms, nil
}

// Get user information by user ID
func (m *postgresDBRepo) GetUserByID(id int) (models.User, error){
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	query := `select id, first_name, last_name, email, password, access_level created_at, updated_at,
				from users where id=$1`
	
	row := m.DB.QueryRowContext(ctx, query, id)

	var u models.User
	err := row.Scan(&u.ID, &u.FirstName, &u.LastName, &u.Email, &u.Password, &u.AccessLevel, &u.CreatedAt, &u.UpdatedAt)
	if err != nil{
		return u, err
	}
	return u, nil
}

// Update user information
func (m *postgresDBRepo) UpdateUser(u models.User) (error){
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	query := `update users set first_name=$1, last_name=$2, email=$3, access_level=$4, updated_at=$5`
	
	_, err := m.DB.ExecContext(ctx, query,u.FirstName,u.LastName,u.Email,u.AccessLevel,time.Now())
	if err != nil{
		return err
	}

	return nil
}

// Get user id by login info
func (m *postgresDBRepo) Authenticate(email, testPassword string) (int, string, error){
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	var id int
	var hashedPassword string

	// to see if this user exist in db or not
	query := `select id, password from users where email=$1`
	row := m.DB.QueryRowContext(ctx, query, email)
	err := row.Scan(&id, &hashedPassword)
	if err != nil{
		return id, hashedPassword, err
	}

	// compare password
	err = bcrypt.CompareHashAndPassword([]byte(hashedPassword),[]byte(testPassword))
	if err == bcrypt.ErrMismatchedHashAndPassword{
		return 0, "", errors.New("Incorrect Password")
	}else if err != nil{
		return 0, "", err
	}

	return id, hashedPassword, nil

}

// Admin - returns a slice of all reservations
func (m *postgresDBRepo) AllReservations() ([]models.Reservations, error){
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	var reservations []models.Reservations

	query := ` select r.id, r.first_name, r.last_name, r.email, r.phone, r.start_date,
				r.end_date, r.room_id, r.created_at, r.updated_at, rm.id, rm.room_name
				from reservations r
				left join rooms rm on (r.room_id = rm.id)
				order by r.start_date asc`

	rows, err := m.DB.QueryContext(ctx, query)
	if err != nil {
		return reservations, err
	}
	defer rows.Close()
	
	for rows.Next(){
		var item models.Reservations
		err := rows.Scan(&item.ID,&item.FirstName,&item.LastName,&item.Email,&item.Phone,&item.StartDate,
			&item.EndDate,&item.RoomID,&item.CreatedAt,&item.UpdatedAt,&item.Room.ID,&item.Room.RoomName)
		if err != nil{
			return reservations, err
		}
		reservations = append(reservations, item)
	}
	return reservations, nil
}
