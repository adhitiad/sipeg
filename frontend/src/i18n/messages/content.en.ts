import type { contentMessagesId } from './content.id'

export const contentMessagesEn = {
  'landing.heroTitle': 'Staff data in one place',
  'landing.heroLead':
    'SIMPEG keeps employee records, unit structure, attendance, leave requests, and pay slips in a single database. Each user only sees the parts they are authorised to see.',
  'landing.primaryCta': 'Sign in to SIMPEG',
  'landing.secondaryCta': 'Read about the system',
  'landing.scopeTitle': 'What SIMPEG keeps',
  'landing.scopeLead':
    'The six groups of data below already have a place in the database schema and the API.',
  'landing.scope.employee.title': 'Employee records',
  'landing.scope.employee.body':
    'Identity, contact details, position history, education history, and salary history.',
  'landing.scope.organization.title': 'Unit structure',
  'landing.scope.organization.body':
    'Work units and their parents and children, used to bound access between units.',
  'landing.scope.attendance.title': 'Attendance',
  'landing.scope.attendance.body':
    'Daily attendance records with the source and location of the entry.',
  'landing.scope.leave.title': 'Leave and permissions',
  'landing.scope.leave.body':
    'Leave requests, supervisor approval, quota balances, and temporary suspension.',
  'landing.scope.payroll.title': 'Payroll',
  'landing.scope.payroll.body':
    'Pay slips per period, salary components, and payment status.',
  'landing.scope.performance.title': 'Performance and tridharma',
  'landing.scope.performance.body':
    'Performance targets, appraisals, and education, research, community service, and support activities.',
  'landing.rolesTitle': 'Who uses SIMPEG',
  'landing.rolesLead': 'Access follows your role. Broader roles hold a wider unit scope.',
  'landing.role.admin.title': 'Administrator',
  'landing.role.admin.body':
    'Manages employee data, organisation, payroll, and the audit trail across the institution.',
  'landing.role.operator.title': 'Operator',
  'landing.role.operator.body':
    'Manages employee data and work units within their assigned unit and its descendants.',
  'landing.role.atasan.title': 'Supervisor',
  'landing.role.atasan.body':
    'Reviews leave requests and oversees employee data within their assigned unit.',
  'landing.role.pegawai.title': 'Employee',
  'landing.role.pegawai.body': 'Sees their own data, pay slips, and leave requests.',
  'landing.accessTitle': 'How access works',
  'landing.accessLead':
    'There is no self-registration. Accounts are created by the institution administrator.',
  'landing.access.account.title': 'Accounts are created by the administrator',
  'landing.access.account.body':
    'The employee email is recorded in the employee data, then an initial password is given by the administrator.',
  'landing.access.role.title': 'Role and unit are assigned',
  'landing.access.role.body':
    'Every account holds one role and one assigned unit. Both determine which modules can be opened.',
  'landing.access.audit.title': 'Important changes are logged',
  'landing.access.audit.body':
    'Changes to employee data and approvals are recorded in the audit trail with the actor and the time.',
  'landing.closeTitle': 'Ready to sign in?',
  'landing.closeBody':
    'Use the staff account given to you by the institution administrator.',
  'landing.closeCta': 'Sign in to SIMPEG',

  'about.title': 'About SIMPEG',
  'about.lead':
    'SIMPEG is the staff information system for institutions that administer employee data centrally.',
  'about.whatTitle': 'What SIMPEG does',
  'about.whatBody':
    'The system stands on three things: one staff database, one access rule based on role and unit, and one audit trail for every important change.',
  'about.whoTitle': 'Who may use it',
  'about.whoBody':
    'Access is decided by role and home unit. Administrators see the whole organisation. Operators and supervisors see their unit and all of its descendants. Employees see their own records. Supported roles: administrator, operator, supervisor, employee.',
  'about.statusTitle': 'Current status',
  'about.statusBody':
    'The backend foundation, database schema, and web interface are available. The web modules for employees, organisation, leave, and payroll are still in progress.',
  'about.accessTitle': 'Requesting access',
  'about.accessBody':
    'Account requests and changes to employee data are filed with the institution administrator. Never send a password by email or an online message.',

  'privacy.title': 'Privacy Policy',
  'privacy.s1t': 'Data that is collected',
  'privacy.s1b':
    'SIMPEG stores the data needed to administer employment: identity, contact details, position and salary history, attendance records, and employment documents.',
  'privacy.s2t': 'How the data is used',
  'privacy.s2b':
    'Data is used solely for employment administration: payroll, attendance, leave, performance appraisal, and audit. It is not sold or shared with third parties without consent.',
  'privacy.s3t': 'Who can see it',
  'privacy.s3b':
    'Access follows role and assigned unit. Administrators can see all data; operators and supervisors are limited to their unit; employees can only see their own records.',
  'privacy.s4t': 'Storage and deletion',
  'privacy.s4b':
    'Data is stored in the institution database. Audit entries are retained for the period approved by the institution and are then deleted by the scheduled cleanup job.',

  'terms.title': 'Terms of Service',
  'terms.s1t': 'Accounts and access',
  'terms.s1b':
    'Accounts are created by the institution administrator. There is no self-registration. Do not share your account credentials with anyone.',
  'terms.s2t': 'Roles and responsibilities',
  'terms.s2b':
    'Administrators manage accounts and access rules. Operators and supervisors act within their assigned unit. Employees are responsible for the accuracy of data they submit.',
  'terms.s3t': 'Acceptable use',
  'terms.s3b':
    'Use SIMPEG only for legitimate employment matters. Do not attempt to access data outside your role or unit, or to interfere with the service.',
  'terms.s4t': 'Changes to these terms',
  'terms.s4b':
    'The institution may update these terms. Material changes will be announced through official channels before they take effect.',

  'reference.title': 'Terms of Reference',
  'reference.lead':
    'This document defines the words used by SIMPEG. It is a draft pending approval by the product owner.',
  'reference.s1t': 'Employee',
  'reference.s1b': 'A person registered in the staff database.',
  'reference.s2t': 'Home unit',
  'reference.s2b':
    'The employee’s primary work unit. Access rights cover the home unit and its descendants in the unit tree.',
  'reference.s3t': 'Role',
  'reference.s3b':
    'A fixed set of permissions attached to an account. Supported roles are administrator, operator, supervisor, and employee.',
  'reference.s4t': 'Audit trail',
  'reference.s4b':
    'An append-only record of important changes: what changed, who changed it, and when.',

  'app.title': 'Workspace',
  'app.signedInAs': 'You are signed in as {email}',
  'app.greeting': 'Hello, {name}',
  'app.accountTitle': 'Your account',
  'app.emailLabel': 'Email',
  'app.roleLabel': 'Role',
  'app.roleMissing': 'Not assigned',
  'app.unitLabel': 'Home unit ID',
  'app.unitMissing': 'Not assigned',
  'app.accessTitle': 'Your access rights',
  'app.accessLead':
    'The list below is sent by the server and follows your role and home unit.',
  'app.accessEmpty':
    'Your role holds no access rights. Contact the institution administrator if this looks wrong.',
  'app.accessCount': '{count} permissions',
  'app.signOut': 'Sign out',

  'legal.draftNotice':
    'Draft document. The content has not been approved by the product owner.',
  'legal.updated': 'Last updated: 12 November 2026',
  'legal.contact':
    'Questions about this document should be filed with the institution administrator.',
} as const satisfies Record<keyof typeof contentMessagesId, string>
