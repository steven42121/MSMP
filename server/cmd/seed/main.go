package main

import (
	"fmt"
	"log"
	"os"

	"MSMP/server/config"
	"MSMP/server/db"
	"MSMP/server/models"

	"golang.org/x/crypto/bcrypt"
)

func main() {
	// 加载配置
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// 初始化数据库
	if err := db.Init(cfg); err != nil {
		log.Fatalf("Failed to init database: %v", err)
	}
	log.Println("Database connected")

	// 创建默认租户
	var tenant models.Tenant
	result := db.DB.Where("slug = ?", "default").First(&tenant)
	if result.Error != nil {
		tenant = models.Tenant{
			Name: "默认租户",
			Slug: "default",
		}
		if err := db.DB.Create(&tenant).Error; err != nil {
			log.Fatalf("Failed to create tenant: %v", err)
		}
		log.Printf("Tenant created: %s (ID=%d)", tenant.Name, tenant.ID)
	} else {
		log.Printf("Tenant already exists: %s (ID=%d)", tenant.Name, tenant.ID)
	}

	// 创建 admin 用户
	var user models.User
	result = db.DB.Where("username = ?", "admin").First(&user)
	if result.Error == nil {
		log.Println("Admin user already exists, updating password...")
	} else {
		user = models.User{
			TenantID: tenant.ID,
			Username: "admin",
			Email:    "admin@msmp.local",
			Role:     "admin",
		}
	}

	// 从环境变量读取密码，未设置则生成随机密码
	password := os.Getenv("MSMP_ADMIN_PASSWORD")
	if password == "" {
		password = generateRandomPassword(16)
		fmt.Println("\n========================================")
		fmt.Println("  MSMP 初始管理员账号（密码仅显示一次！）")
		fmt.Printf("  用户名: admin\n  密码:   %s\n", password)
		fmt.Println("========================================")
		fmt.Println("  提示: 设置环境变量 MSMP_ADMIN_PASSWORD 可指定密码")
		fmt.Println("========================================")
	}

	// 加密密码
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		log.Fatalf("Failed to hash password: %v", err)
	}
	user.PasswordHash = string(hash)
	user.MustChangePassword = true

	if result.Error == nil {
		db.DB.Model(&user).Updates(map[string]interface{}{
			"password_hash":       user.PasswordHash,
			"must_change_password": true,
		})
	} else {
		if err := db.DB.Create(&user).Error; err != nil {
			log.Fatalf("Failed to create user: %v", err)
		}
	}

	log.Printf("Admin user ready (must_change_password=true)")
}

func generateRandomPassword(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789!@#$%^&*"
	b := make([]byte, length)
	for i := range b {
		n, _ := randomInt(len(charset))
		b[i] = charset[n]
	}
	return string(b)
}

func randomInt(max int) (int, error) {
	b := make([]byte, 1)
	if _, err := randomBytes(b); err != nil {
		return 0, err
	}
	return int(b[0]) % max, nil
}

func randomBytes(b []byte) (int, error) {
	f, err := os.Open("/dev/urandom")
	if err != nil {
		return 0, err
	}
	defer f.Close()
	return f.Read(b)
}
