package main

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"github.com/google/uuid"
	"os"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

type User struct {
	ID        string    `db:"id"`
	Name      string    `db:"name"`
	Email     string    `db:"email"`
	CreatedAt time.Time `db:"created_at"`
}

type Blob struct {
	ID          string    `db:"id"`
	Key         string    `db:"key"`
	Filename    string    `db:"filename"`
	ContentType string    `db:"content_type"`
	ByteSize    int64     `db:"byte_size"`
	Checksum    string    `db:"checksum"`
	CreatedAt   time.Time `db:"created_at"`
}

type Attachment struct {
	ID         string    `db:"id"`
	Name       string    `db:"name"`
	RecordType string    `db:"record_type"`
	RecordID   string    `db:"record_id"`
	BlobID     string    `db:"blob_id"`
	CreatedAt  time.Time `db:"created_at"`
}

var db *sqlx.DB

func main() {
	app := fiber.New()

	var err error
	db, err = sqlx.Connect("postgres", "")
	if err != nil {
		panic(err)
	}

	app.Post("/users/:id/avatar", uploadUserAvatar)
	app.Get("/users/:id/avatar", serveUserAvatar)
	app.Listen(":3000")
}

func uploadUserAvatar(c *fiber.Ctx) error {
	userID := c.Params("id")

	file, err := c.FormFile("avatar")
	if err != nil {
		return c.Status(400).SendString("No file uploaded")
	}

	// Generate a unique key for the file
	fileKey := generateFileKey(file.Filename)
	filePath := fmt.Sprintf("./uploads/%s", fileKey)

	if err := c.SaveFile(file, filePath); err != nil {
		return c.Status(500).SendString("Could not save file")
	}

	// Calculate file checksum
	checksum := calculateChecksum(filePath)

	blobId := uuid.NewString()

	// Save the blob metadata in the database
	blob := Blob{
		ID:          blobId,
		Key:         fileKey,
		Filename:    file.Filename,
		ContentType: file.Header.Get("Content-Type"),
		ByteSize:    file.Size,
		Checksum:    checksum,
		CreatedAt:   time.Now(),
	}

	_, err = db.NamedExec(`INSERT INTO blobs (id,key, filename, content_type, byte_size, checksum, created_at)
                                VALUES (:id, :key, :filename, :content_type, :byte_size, :checksum, :created_at)`, &blob)
	if err != nil {
		return c.Status(500).SendString("Could not save blob metadata")
	}

	// Associate the blob with the user
	err = attachFile("User", userID, blob.ID, "avatar")
	if err != nil {
		return c.Status(500).SendString("Could not associate file with user")
	}

	return c.JSON(blob)
}

func generateFileKey(filename string) string {
	return fmt.Sprintf("%s_%d", filename, time.Now().UnixNano())
}

func calculateChecksum(filePath string) string {
	file, err := os.Open(filePath)
	if err != nil {
		return ""
	}
	defer file.Close()

	hash := md5.New()
	if _, err := hash.Write([]byte(filePath)); err != nil {
		return ""
	}

	return hex.EncodeToString(hash.Sum(nil))
}

func attachFile(recordType string, recordID string, blobID string, name string) error {
	attachment := Attachment{
		ID:         uuid.NewString(),
		Name:       name,
		RecordType: recordType,
		RecordID:   recordID,
		BlobID:     blobID,
		CreatedAt:  time.Now(),
	}

	_, err := db.NamedExec(`INSERT INTO attachments (id, name, record_type, record_id, blob_id, created_at)
                           VALUES (:id, :name, :record_type, :record_id, :blob_id, :created_at)`, &attachment)
	fmt.Println(err)
	return err
}

func serveUserAvatar(c *fiber.Ctx) error {
	userID := c.Params("id")

	var blob Blob
	err := db.Get(&blob, `
        SELECT blobs.* FROM blobs
        JOIN attachments ON attachments.blob_id = blobs.id
        WHERE attachments.record_type = 'User' AND attachments.record_id = $1 AND attachments.name = 'avatar'
    `, userID)
	if err != nil {
		return c.Status(404).SendString("Avatar not found")
	}

	filePath := fmt.Sprintf("./uploads/%s", blob.Key)

	recalculatedChecksum := calculateChecksum(filePath)

	// Compare the recalculated checksum with the stored checksum
	if recalculatedChecksum != blob.Checksum {
		return c.Status(500).SendString("File checksum validation failed, file may be corrupted")
	}

	return c.SendFile(filePath)
}
