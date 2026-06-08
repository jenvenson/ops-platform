package platformobject

import (
	"fmt"
	"strings"

	"github.com/edy/ops-platform/internal/cmdb"
	"github.com/edy/ops-platform/internal/database"
	"github.com/edy/ops-platform/internal/models"
	"gorm.io/gorm"
)

const (
	ObjectTypeProject      = "project"
	ObjectTypeApplication  = "application"
	ObjectTypeDeployRecord = "deploy_record"

	sourceModuleCMDB = "cmdb"
)

func Init() error {
	if err := database.DB.AutoMigrate(&models.PlatformObject{}); err != nil {
		return fmt.Errorf("failed to migrate platform_object_index: %w", err)
	}
	if err := SyncSeedData(database.DB); err != nil {
		return fmt.Errorf("failed to sync platform object seed data: %w", err)
	}
	return nil
}

func SyncSeedData(db *gorm.DB) error {
	if db == nil {
		return fmt.Errorf("nil db")
	}

	var projects []cmdb.Project
	if err := db.Where("deleted_at IS NULL").Find(&projects).Error; err != nil {
		return err
	}
	for _, project := range projects {
		if err := upsertObject(db, buildProjectObject(project)); err != nil {
			return err
		}
	}

	var applications []cmdb.Application
	if err := db.Preload("Project").Preload("Environment").Where("deleted_at IS NULL").Find(&applications).Error; err != nil {
		return err
	}
	for _, application := range applications {
		if err := upsertObject(db, buildApplicationObject(application)); err != nil {
			return err
		}
	}

	var deployRecords []cmdb.DeployRecord
	if err := db.Order("id desc").Limit(1000).Find(&deployRecords).Error; err != nil {
		return err
	}
	for _, record := range deployRecords {
		if err := upsertObject(db, buildDeployRecordObject(record)); err != nil {
			return err
		}
	}

	return nil
}

func buildProjectObject(project cmdb.Project) models.PlatformObject {
	return models.PlatformObject{
		ObjectUID:    objectUID(ObjectTypeProject, sourceModuleCMDB, project.ID),
		ObjectType:   ObjectTypeProject,
		SourceModule: sourceModuleCMDB,
		SourcePK:     uintString(project.ID),
		Title:        strings.TrimSpace(project.Name),
		Summary:      projectSummary(project),
		Status:       objectStatus(project.DeletedAt == nil, "", ""),
		MetadataJSON: mustObjectMetadataJSON(map[string]any{
			"code":        strings.TrimSpace(project.Code),
			"description": strings.TrimSpace(project.Description),
		}),
	}
}

func buildApplicationObject(app cmdb.Application) models.PlatformObject {
	projectName := ""
	if app.Project != nil {
		projectName = strings.TrimSpace(app.Project.Name)
	}
	envName := ""
	if app.Environment != nil {
		envName = strings.TrimSpace(app.Environment.Name)
	}

	return models.PlatformObject{
		ObjectUID:    objectUID(ObjectTypeApplication, sourceModuleCMDB, app.ID),
		ObjectType:   ObjectTypeApplication,
		SourceModule: sourceModuleCMDB,
		SourcePK:     uintString(app.ID),
		Title:        strings.TrimSpace(app.Name),
		Summary:      applicationSummary(app, projectName, envName),
		Status:       objectStatus(app.DeletedAt == nil, "", ""),
		MetadataJSON: mustObjectMetadataJSON(map[string]any{
			"projectId":   app.ProjectID,
			"projectName": projectName,
			"envId":       app.EnvID,
			"envName":     envName,
			"jenkinsJob":  strings.TrimSpace(app.JenkinsJob),
		}),
	}
}

func buildDeployRecordObject(record cmdb.DeployRecord) models.PlatformObject {
	return models.PlatformObject{
		ObjectUID:    objectUID(ObjectTypeDeployRecord, sourceModuleCMDB, record.ID),
		ObjectType:   ObjectTypeDeployRecord,
		SourceModule: sourceModuleCMDB,
		SourcePK:     uintString(record.ID),
		Title:        deployRecordTitle(record),
		Summary:      deployRecordSummary(record),
		Status:       strings.TrimSpace(record.Status),
		OwnerID:      strings.TrimSpace(record.TriggeredBy),
		MetadataJSON: mustObjectMetadataJSON(map[string]any{
			"appId":       record.AppID,
			"appName":     strings.TrimSpace(record.AppName),
			"envId":       record.EnvID,
			"envName":     strings.TrimSpace(record.EnvName),
			"projectCode": strings.TrimSpace(record.ProjectCode),
			"deployType":  strings.TrimSpace(record.DeployType),
		}),
	}
}

func upsertObject(db *gorm.DB, object models.PlatformObject) error {
	var existing models.PlatformObject
	err := db.Where("object_uid = ?", object.ObjectUID).First(&existing).Error
	if err == nil {
		return db.Model(&existing).Updates(map[string]any{
			"object_type":   object.ObjectType,
			"source_module": object.SourceModule,
			"source_pk":     object.SourcePK,
			"title":         object.Title,
			"summary":       object.Summary,
			"status":        object.Status,
			"owner_id":      object.OwnerID,
			"metadata_json": object.MetadataJSON,
		}).Error
	}
	if err != nil && err != gorm.ErrRecordNotFound {
		return err
	}
	return db.Create(&object).Error
}
