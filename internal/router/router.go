package router

import (
	"time"

	"github.com/gofiber/fiber/v2"

	"github.com/noah-isme/gema-go-api/internal/config"
	"github.com/noah-isme/gema-go-api/internal/handler"
	"github.com/noah-isme/gema-go-api/internal/middleware"
)

// Dependencies groups router dependencies for registration.
type Dependencies struct {
	AssignmentHandler        *handler.AssignmentHandler
	SubmissionHandler        *handler.SubmissionHandler
	TutorialContentHandler   *handler.TutorialContentHandler
	RoadmapHandler           *handler.RoadmapHandler
	StudentDashboardHandler  *handler.StudentDashboardHandler
	WebLabHandler            *handler.WebLabHandler
	CodingTaskHandler        *handler.CodingTaskHandler
	CodingSubmissionHandler  *handler.CodingSubmissionHandler
	AdminStudentHandler      *handler.AdminStudentHandler
	AdminAssignmentHandler   *handler.AdminAssignmentHandler
	AdminGradingHandler      *handler.AdminGradingHandler
	AdminAnalyticsHandler    *handler.AdminAnalyticsHandler
	AdminActivityHandler     *handler.AdminActivityHandler
	AdminContactHandler      *handler.AdminContactHandler
	AdminGalleryHandler      *handler.AdminGalleryHandler
	AdminAnnouncementHandler *handler.AdminAnnouncementHandler
	ChatHandler              *handler.ChatHandler
	NotificationHandler      *handler.NotificationHandler
	DiscussionHandler        *handler.DiscussionHandler
	ActivityFeedHandler      *handler.ActivityFeedHandler
	AnnouncementHandler      *handler.AnnouncementHandler
	GalleryHandler           *handler.GalleryHandler
	ContactHandler           *handler.ContactHandler
	UploadHandler            *handler.UploadHandler
	SeedHandler              *handler.SeedHandler
	JWTMiddleware            fiber.Handler
}

// Register wires the HTTP routes into the fiber application.
func Register(app *fiber.App, cfg config.Config, deps Dependencies) {
	// Common v1 group for health & headers
	api := app.Group("/api/v1", func(c *fiber.Ctx) error {
		c.Set("X-Application", cfg.AppName)
		return c.Next()
	})
	api.Get("/health", handler.HealthCheck(cfg))

	// Use provided JWT middleware, or a no-op if nil
	jwtMiddleware := deps.JWTMiddleware
	if jwtMiddleware == nil {
		jwtMiddleware = func(c *fiber.Ctx) error { return c.Next() }
	}

	// Tutorial (assignments & submissions)
	if deps.AssignmentHandler != nil {
		tutorial := app.Group("/api/v2/tutorial", jwtMiddleware)
		assignmentGroup := tutorial.Group("/assignments")
		deps.AssignmentHandler.Register(assignmentGroup)

		if deps.SubmissionHandler != nil {
			submissionGroup := tutorial.Group("/submissions")
			deps.SubmissionHandler.Register(submissionGroup)
		}
	}

	if deps.TutorialContentHandler != nil {
		publicTutorial := app.Group("/api/tutorial")
		deps.TutorialContentHandler.RegisterPublic(publicTutorial)

		adminTutorial := app.Group("/api/tutorial", jwtMiddleware, middleware.RequireRole("admin", "teacher"))
		deps.TutorialContentHandler.RegisterAdmin(adminTutorial)
	}

	if deps.RoadmapHandler != nil {
		roadmap := app.Group("/api/roadmap")
		deps.RoadmapHandler.Register(roadmap)
	}

	// Web Lab
	if deps.WebLabHandler != nil {
		webLab := app.Group("/api/v2/web-lab", jwtMiddleware)
		deps.WebLabHandler.Register(webLab)
	}

	// Coding Lab (tasks & submissions)
	if deps.CodingTaskHandler != nil {
		codingLab := app.Group("/api/v2/coding-lab", jwtMiddleware)

		taskGroup := codingLab.Group("/tasks")
		deps.CodingTaskHandler.Register(taskGroup)

		if deps.CodingSubmissionHandler != nil {
			submissionGroup := codingLab.Group("/submissions")
			deps.CodingSubmissionHandler.Register(submissionGroup)
		}
	}

	// Student dashboard
	if deps.StudentDashboardHandler != nil {
		student := app.Group("/api/v2/student", jwtMiddleware)
		deps.StudentDashboardHandler.Register(student)
	}

	if deps.ChatHandler != nil {
		chat := app.Group("/api/v2/chat", jwtMiddleware, middleware.RequireRole("student", "teacher", "admin"), middleware.RateLimit("chat", 10, time.Second))
		deps.ChatHandler.Register(chat)
	}

	if deps.NotificationHandler != nil {
		notifications := app.Group("/api/v2/notifications", jwtMiddleware, middleware.RequireRole("student", "teacher", "admin"), middleware.RateLimit("notifications", 8, time.Second))
		deps.NotificationHandler.Register(notifications)
	}

	if deps.DiscussionHandler != nil {
		discussions := app.Group("/api/v2/discussion", jwtMiddleware, middleware.RequireRole("student", "teacher", "admin"), middleware.RateLimit("discussion", 20, time.Second))
		deps.DiscussionHandler.Register(discussions)
	}

	if deps.AdminStudentHandler != nil || deps.AdminAssignmentHandler != nil || deps.AdminGradingHandler != nil || deps.AdminAnalyticsHandler != nil || deps.AdminActivityHandler != nil || deps.AdminContactHandler != nil || deps.AdminGalleryHandler != nil || deps.AdminAnnouncementHandler != nil {
		admin := app.Group("/api/admin", jwtMiddleware, middleware.RequireRole("admin", "teacher"))

		if deps.AdminStudentHandler != nil {
			studentGroup := admin.Group("/students")
			deps.AdminStudentHandler.Register(studentGroup)
		}

		if deps.AdminAssignmentHandler != nil {
			assignmentGroup := admin.Group("/assignments")
			deps.AdminAssignmentHandler.Register(assignmentGroup)
		}

		if deps.AdminGradingHandler != nil {
			submissionGroup := admin.Group("/submissions")
			deps.AdminGradingHandler.Register(submissionGroup)
		}

		if deps.AdminAnalyticsHandler != nil {
			analyticsGroup := admin.Group("/analytics")
			deps.AdminAnalyticsHandler.Register(analyticsGroup)
		}

		if deps.AdminActivityHandler != nil {
			activityGroup := admin.Group("/activities")
			deps.AdminActivityHandler.Register(activityGroup)
		}

		if deps.AdminContactHandler != nil {
			contactGroup := admin.Group("/contacts")
			deps.AdminContactHandler.Register(contactGroup)
		}

		if deps.AdminGalleryHandler != nil {
			galleryGroup := admin.Group("/gallery")
			deps.AdminGalleryHandler.Register(galleryGroup)
		}
		if deps.AdminAnnouncementHandler != nil {
			announcementGroup := admin.Group("/announcements")
			deps.AdminAnnouncementHandler.Register(announcementGroup)
		}
	}
	if deps.ActivityFeedHandler != nil {
		activities := app.Group("/api/activities")
		deps.ActivityFeedHandler.Register(activities)
	}

	if deps.AnnouncementHandler != nil {
		announcements := app.Group("/api/announcements")
		deps.AnnouncementHandler.Register(announcements)
	}

	if deps.GalleryHandler != nil {
		gallery := app.Group("/api/gallery")
		deps.GalleryHandler.Register(gallery)
	}

	if deps.ContactHandler != nil {
		contact := app.Group("/api/contact", jwtMiddleware, middleware.RateLimit("contact", 5, time.Minute))
		deps.ContactHandler.Register(contact)
	}

	if deps.UploadHandler != nil {
		upload := app.Group("/api/upload", jwtMiddleware, middleware.RequireRole("student", "teacher", "admin"), middleware.RateLimit("upload", 3, time.Minute))
		deps.UploadHandler.Register(upload)
	}

	if deps.SeedHandler != nil {
		seed := app.Group("/api/seed", jwtMiddleware, middleware.RequireRole("admin"), middleware.RateLimit("seed", 1, time.Minute))
		deps.SeedHandler.Register(seed)
	}

}
