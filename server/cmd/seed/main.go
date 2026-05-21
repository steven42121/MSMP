package main

import (
	"fmt"
	"log"

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

	// 加密密码
	hash, err := bcrypt.GenerateFromPassword([]byte("admin123"), bcrypt.DefaultCost)
	if err != nil {
		log.Fatalf("Failed to hash password: %v", err)
	}
	user.PasswordHash = string(hash)

	if result.Error == nil {
		db.DB.Model(&user).Update("password_hash", user.PasswordHash)
	} else {
		if err := db.DB.Create(&user).Error; err != nil {
			log.Fatalf("Failed to create user: %v", err)
		}
	}

	log.Printf("Admin user ready: username=admin, password=admin123")
	fmt.Println("\n========================================")
	fmt.Println("  MSMP 初始管理员账号")
	fmt.Println("  用户名: admin")
	fmt.Println("  密码:   admin123")
	fmt.Println("========================================")
}