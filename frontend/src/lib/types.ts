export interface User {
  id: number
  name: string
  email: string
  role: string
  createdAt: string
  updatedAt: string
}

export interface AuthResponse {
  token: string
  user: User
  onboarded: boolean
}

export interface RegisterResponse {
  email: string
  verificationRequired: boolean
}

export interface EducationLevel {
  id: number
  name: string
  code: string
  sequence: number
}

export interface ClassLevel {
  id: number
  name: string
  code: string
  sequence: number
  educationLevelId: number
}

export interface Term {
  id: number
  name: string
  code: string
  sequence: number
}

export interface Subject {
  id: number
  name: string
  code: string
}

export interface StudentProfile {
  id: number
  userId: number
  educationLevelId?: number
  classLevelId?: number
  educationLevel?: EducationLevel
  classLevel?: ClassLevel
  createdAt: string
  updatedAt: string
}

export interface ProfileResponse {
  profile: StudentProfile
  onboarded: boolean
}

export type MasteryState = 'STRONG' | 'IMPROVING' | 'WEAK' | 'NEW' | ''

export interface TopicView {
  id: number
  name: string
  description: string
  estimatedMinutes: number
  classLevelId: number
  termId: number
  subjectId: number
  hasLesson: boolean
  hasQuestions: boolean
  mastery?: number
  state?: MasteryState
}

export interface TopicDetail extends TopicView {
  subject: Subject
  classLevel: ClassLevel
  term: Term
  lessonId?: number
  questionCount: number
}

export interface LessonExample {
  title: string
  problem: string
  solution: string
}

export interface LessonFormula {
  name: string
  expression: string
  meaning: string
}

export interface Lesson {
  id: number
  topicId: number
  title: string
  introduction: string
  explanation: string
  examples: LessonExample[]
  keyPoints: string[]
  formulas: LessonFormula[]
  topicName: string
  subjectName: string
  className: string
  termName: string
  createdAt: string
}

export interface QuestionOption {
  id: number
  key: string
  text: string
}

export interface Question {
  id: number
  topicId: number
  type: 'MCQ' | 'CALCULATION'
  prompt: string
  difficulty: number
  marks: number
  options: QuestionOption[]
}

export interface StartQuizResponse {
  quizId: number
  topic: TopicView
  questions: Question[]
  totalMarks: number
}

export interface SubmitAnswer {
  questionId: number
  selectedOptionId?: number
  answer?: string
  working?: string
}

export interface SubmitResult {
  quizId: number
  score: number
  scoreMarks: number
  correctCount: number
  totalCount: number
  status: string
}

export interface AnswerFeedback {
  question: Question
  isCorrect: boolean
  receivedMarks: number
  studentAnswer: string
  working: string
  selectedOptionId?: number
  explanation: string
  expectedWorking: string
  correctAnswer: string
  ai: {
    status: '' | 'PENDING' | 'SUCCESS' | 'FALLBACK' | 'FAILED'
    feedback?: Record<string, unknown> | null
    kind?: string
  }
}

export interface QuizResultResponse {
  quiz: {
    id: number
    topicId: number
    status: string
    isRecovery: boolean
    totalMarks: number
    scoreMarks: number
    score: number
    correctCount: number
    totalCount: number
  }
  topic: TopicView
  subject: Subject
  className: string
  term: Term
  feedback: AnswerFeedback[]
}

export interface KnowledgeEntry {
  id: number
  name: string
  state: MasteryState
  mastery: number
  attempts: number
  correct: number
  className: string
  termName: string
}

export interface KnowledgeGroup {
  subject: Subject
  topics: KnowledgeEntry[]
  total: number
  mastered: number
  improving: number
  weak: number
  percentage: number
}

export interface SubjectProgress {
  id: number
  subject?: Subject
  topicsTotal: number
  topicsMastered: number
  correctCount: number
  totalCount: number
  percentage: number
  lastAssessedAt?: string
}

export interface GamificationSummary {
  xp: number
  level: number
  streak: number
  longestStreak: number
  badges: Array<{ badge: { name: string; description: string }; earnedAt: string }>
}

export interface BadgeRow {
  id: number
  code: string
  name: string
  description: string
  earned: boolean
}

export interface Challenge {
  id: number
  code: string
  title: string
  description: string
  metric: string
  target: number
  xpReward: number
  active: boolean
  progress: number
  completed: boolean
}

export interface Recommendation {
  id: number
  kind: 'WEAK' | 'RECOVERY'
  priority: number
  reason: string
  status: string
  createdAt: string
  topic: {
    id: number
    name: string
    description: string
    subject: Subject
    className: string
    term: Term
    mastery: number
    state: MasteryState
    lessonId?: number
  }
  steps: string
}

export interface RecoveryResponse {
  missions: Recommendation[]
  steps: string
}

export interface MistakeFeedback {
  correct: boolean
  mistakeType: string
  whatStudentDid: string
  conceptUnderstood: string
  explanation: string
  correctWorking: string
  recommendedAction: string
}

export interface WorkingStep {
  label: string
  status: string
  comment: string
}

export interface WorkingFeedback {
  steps: WorkingStep[]
  overallFeedback: string
  correctSolution: string
}

export interface ExplainFeedback {
  title: string
  explanation: string
  keyPoints: string[]
  example: string
}

export interface PracticeQuestion {
  prompt: string
  options: Array<{ key: string; text: string }>
  correctKey: string
  explanation: string
  difficulty: number
}

export interface AIEnvelope<T> {
  data: T
  aiStatus: string
  error: string
}

export interface AdminRecentUser {
  id: number
  name: string
  email: string
  role: string
  emailVerified: boolean
  createdAt: string
}

export interface AdminOverview {
  totalUsers: number
  verifiedUsers: number
  onboardedUsers: number
  totalQuizzes: number
  submittedQuizzes: number
  totalAnswers: number
  aiAnalyses: number
  activeStudentsToday: number
  avgScore: number
  recentRegistrations: AdminRecentUser[]
}

export interface AdminUserRow {
  id: number
  name: string
  email: string
  role: string
  emailVerified: boolean
  onboarded: boolean
  quizCount: number
  submittedCount: number
  avgScore: number
  xp: number
  masteredTopics: number
  lastActiveAt?: string
  createdAt: string
}

export interface LibraryNote {
  id: number
  subject: string
  topic: string
  title: string
  content: string
  createdAt: string
}

export interface LibrarySubject {
  name: string
  notes: LibraryNote[]
  count: number
}
