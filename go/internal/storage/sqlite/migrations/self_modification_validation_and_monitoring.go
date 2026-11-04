package migrations

// SelfModificationValidationAndMonitoring adds validation plan, verification results,
// and post-deployment monitoring metrics to self_modification_proposals (AIA-14 §4.6, §4.10, §7.1).
const SelfModificationValidationAndMonitoring = `
ALTER TABLE self_modification_proposals ADD COLUMN validation_plan_json TEXT NOT NULL DEFAULT '[]';
ALTER TABLE self_modification_proposals ADD COLUMN verification_results_json TEXT NOT NULL DEFAULT '[]';
ALTER TABLE self_modification_proposals ADD COLUMN post_deploy_metrics_json TEXT NOT NULL DEFAULT '{}';
`
