# Specification: User Article Saving Feature

## 1. Goal

Allow authenticated users to save articles to a "Read Later" list for future reading.

## 2. User Stories

- **As a user**, I want to click a "Save" button on an article page to save it.
- **As a user**, I want to view my saved articles in a dedicated "Read Later" dashboard.
- **As a user**, I want to remove articles from my "Read Later" list.

## 3. Technical Requirements

- **API Endpoint**: `POST /api/articles/{id}/save` (Auth required)
- **Data Model**: `SavedArticle` (id, user_id, article_id, saved_at)
- **UI Components**:
  - Add "Save" button to `ArticleView.jsx`.
  - Create `SavedArticlesPage.jsx`.
- **Database**: Add `saved_articles` table to PostgreSQL.

## 4. Acceptance Criteria (Given/When/Then)

- **Scenario**: User saves an article
  - **Given** I am logged in and viewing an article
  - **When** I click "Save"
  - **Then** the button changes to "Saved"
  - **And** the article is added to my "Read Later" list in the database.

## 5. Edge Cases

- Cannot save the same article twice.
- Unauthenticated users are prompted to log in when clicking "Save".
