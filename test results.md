PS C:\Users\LKsnj\Desktop\RESEARCH_LAKSHAN\IMPLIMENTATION\low-code-workflow-engine\backend> qawawccc^C
PS C:\Users\LKsnj\Desktop\RESEARCH_LAKSHAN\IMPLIMENTATION\low-code-workflow-engine\backend> cd "C:\Users\LKsnj\Desktop\RESEARCH_LAKSHAN\IMPLIMENTATION\low-code-workflow-engine\backend\tests"
PS C:\Users\LKsnj\Desktop\RESEARCH_LAKSHAN\IMPLIMENTATION\low-code-workflow-engine\backend\tests> go test ./... -count=1 -v
=== RUN   TestChatEndpointWithMockEmbeddingSearchAndGemini
--- PASS: TestChatEndpointWithMockEmbeddingSearchAndGemini (0.01s)
PASS
ok      github.com/sanjeewa/agentic-orchestrator/tests/integration      0.145s
?       github.com/sanjeewa/agentic-orchestrator/tests/mocks    [no test files]
=== RUN   TestDatasetLoaderLoadsToolsRulesTemplatesAndExamples
--- PASS: TestDatasetLoaderLoadsToolsRulesTemplatesAndExamples (0.11s)
=== RUN   TestSemanticSearchRetrievesProcurementTools
--- PASS: TestSemanticSearchRetrievesProcurementTools (0.00s)
=== RUN   TestExternalSemanticSearchClientUsesMockEmbeddingService
--- PASS: TestExternalSemanticSearchClientUsesMockEmbeddingService (0.01s)
=== RUN   TestPromptBuilderIncludesRetrievedContextAndExamples
--- PASS: TestPromptBuilderIncludesRetrievedContextAndExamples (0.00s)
=== RUN   TestGeminiCandidateParserExtractsCandidateBlocks
--- PASS: TestGeminiCandidateParserExtractsCandidateBlocks (0.00s)
=== RUN   TestRegistryValidatorBlocksUnknownAction
--- PASS: TestRegistryValidatorBlocksUnknownAction (0.00s)
=== RUN   TestRegistryValidatorBlocksMissingVendorID
--- PASS: TestRegistryValidatorBlocksMissingVendorID (0.00s)
=== RUN   TestRegistryValidatorBlocksEmployeeFinanceClearInvoice
--- PASS: TestRegistryValidatorBlocksEmployeeFinanceClearInvoice (0.00s)
=== RUN   TestRegistryValidatorBlocksClearInvoiceBeforeGoodsReceipt
--- PASS: TestRegistryValidatorBlocksClearInvoiceBeforeGoodsReceipt (0.00s)
=== RUN   TestCandidateSelectorChoosesHighestScoringPassedCandidate
--- PASS: TestCandidateSelectorChoosesHighestScoringPassedCandidate (0.00s)
=== RUN   TestChatOrchestrationReturnsCanExecuteFalseWhenAllCandidatesFail
--- PASS: TestChatOrchestrationReturnsCanExecuteFalseWhenAllCandidatesFail (0.00s)
=== RUN   TestRegistryValidatorBlocksSensitiveParameters
--- PASS: TestRegistryValidatorBlocksSensitiveParameters (0.00s)
=== RUN   TestRunnerExecutesRegisteredTool
--- PASS: TestRunnerExecutesRegisteredTool (0.00s)
=== RUN   TestSemanticSearchGeneratedAccuracyReport
    semantic_and_generation_accuracy_test.go:104: semantic search report: C:\Users\LKsnj\Desktop\RESEARCH_LAKSHAN\IMPLIMENTATION\low-code-workflow-engine\backend\test-results\semantic_search_accuracy_report.html
    semantic_and_generation_accuracy_test.go:105: semantic search metrics: loaded_tools=280 loaded_rules=300 total=1000 accuracy=0.857 tool_recall=0.935 rule_recall=1.000 top1=0.643 mrr=0.714
--- PASS: TestSemanticSearchGeneratedAccuracyReport (77.18s)
=== RUN   TestGeminiGenerationGeneratedLongFlowAccuracyReport
    semantic_and_generation_accuracy_test.go:168: gemini generation report: C:\Users\LKsnj\Desktop\RESEARCH_LAKSHAN\IMPLIMENTATION\low-code-workflow-engine\backend\test-results\gemini_generation_5000_report.html
    semantic_and_generation_accuracy_test.go:169: gemini generation dataset: C:\Users\LKsnj\Desktop\RESEARCH_LAKSHAN\IMPLIMENTATION\low-code-workflow-engine\backend\test-results\gemini_generation_5000_flows.jsonl
    semantic_and_generation_accuracy_test.go:170: gemini generation metrics: total=5000 accuracy=0.950 precision=0.917 recall=1.000 specificity=0.889 f1=0.957 mcc=0.903
--- PASS: TestGeminiGenerationGeneratedLongFlowAccuracyReport (2.20s)
=== RUN   TestGeminiLiveAPIGenerationAccuracyReport
    semantic_and_generation_accuracy_test.go:180: set RUN_GEMINI_LIVE_TEST=1 and GEMINI_API_KEY to run the live Gemini API accuracy check
--- SKIP: TestGeminiLiveAPIGenerationAccuracyReport (0.00s)
=== RUN   TestSynthesizerFallbackReturnsYAML
--- PASS: TestSynthesizerFallbackReturnsYAML (0.00s)
=== RUN   TestRegistryValidatorAccuracyReport
    validator_accuracy_test.go:68: validator accuracy report: C:\Users\LKsnj\Desktop\RESEARCH_LAKSHAN\IMPLIMENTATION\low-code-workflow-engine\backend\test-results\validator_accuracy_report.html
    validator_accuracy_test.go:69: validator metrics: accuracy=1.000 precision=1.000 recall=1.000 specificity=1.000 f1=1.000 mcc=1.000
    validator_accuracy_test.go:71:
        accuracy  | ######################## 1.000
        precision | ######################## 1.000
        recall    | ######################## 1.000
        f1        | ######################## 1.000
        mcc       | ######################## 1.000
--- PASS: TestRegistryValidatorAccuracyReport (0.06s)
=== RUN   TestRegistryValidatorGeneratedLongFlowAccuracyReport
    validator_accuracy_test.go:98: generated flow dataset: C:\Users\LKsnj\Desktop\RESEARCH_LAKSHAN\IMPLIMENTATION\low-code-workflow-engine\backend\test-results\validator_generated_5000_flows.jsonl
    validator_accuracy_test.go:99: generated validator accuracy report: C:\Users\LKsnj\Desktop\RESEARCH_LAKSHAN\IMPLIMENTATION\low-code-workflow-engine\backend\test-results\validator_generated_5000_report.html
    validator_accuracy_test.go:100: generated validator metrics: total=5000 accuracy=0.950 precision=0.917 recall=1.000 specificity=0.889 f1=0.957 mcc=0.903
    validator_accuracy_test.go:102:
        accuracy  | #######################  0.950
        precision | ######################   0.917
        recall    | ######################## 1.000
        f1        | #######################  0.957
        mcc       | ######################   0.903
--- PASS: TestRegistryValidatorGeneratedLongFlowAccuracyReport (1.05s)
=== RUN   TestValidatorAcceptsSafeWorkflow
--- PASS: TestValidatorAcceptsSafeWorkflow (0.00s)
=== RUN   TestValidatorBlocksDirectDatabaseAccess
--- PASS: TestValidatorBlocksDirectDatabaseAccess (0.00s)
PASS
ok      github.com/sanjeewa/agentic-orchestrator/tests/unit     81.783s
PS C:\Users\LKsnj\Desktop\RESEARCH_LAKSHAN\IMPLIMENTATION\low-code-workflow-engine\backend\tests>