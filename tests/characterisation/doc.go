// Package characterisation pins the CURRENT behaviour of the steps that
// pkg/usecase/project.SetupProjectUseCase.Execute performs, so that GL0's
// collapse of the two orchestrators cannot change any of it by accident.
//
// These are characterisation tests, not specifications. Where today's behaviour
// is wrong, the test asserts the wrong behaviour and says so - the point is that
// changing it becomes a visible, deliberate diff rather than a silent one. The
// clearest case is the security scan: it warns and proceeds, which GL0 T9
// replaces with a fail-closed gate. The assertions below pin the warn-only
// behaviour precisely so T9's change shows up as an intentional edit to this
// file.
//
// TRACEABILITY. Every test names the step it covers and cites the line in
// pkg/usecase/project/setup_project.go that performs it, as of commit d23d4e1:
//
//	Step 1  parse            setup_project.go:105
//	Step 2  AI analysis      setup_project.go:173 (deferred) / :289 (kicked off)
//	Step 3  clone            setup_project.go:193
//	Step 4  security scan    setup_project.go:310   (warn-only at :318, :322)
//	Step 5  analyse          setup_project.go:346
//	Step 6  execute+auto-fix setup_project.go:414
//	Step 7  ready            setup_project.go:501
//
// SCOPE. Steps 1, 3, 4 and 5 are covered here by driving the same components
// Execute drives, constructed directly.
//
// Steps 2, 6, 7 and Execute's end-to-end ordering are covered separately, in
// pkg/usecase/project/characterisation_test.go, which drives Execute itself
// through the container option seam added at T1b. The auto-fix loop is covered in
// pkg/infrastructure/executor, where it actually lives.
//
// NO NETWORK, NO AI. The clone tests use a git repository created in a temp
// directory and cloned from a local path, so `git clone` runs for real but
// never leaves the machine. Nothing here contacts an AI provider.
package characterisation
