-- Questionnaire version integrity (ztmf-misc#394); run by
-- `make check-questionnaire-versions`. Prints one line per failing check and
-- nothing when clean. The orphan counts mirror 0068's ZTMF394_ORPHANED_CATALOG
-- guard so they can run before 0068 is applied.
SELECT 'orphaned_functions', count(*)
  FROM public.functions f
 WHERE f.questionid IS NOT NULL
   AND NOT EXISTS (SELECT 1 FROM public.questions q WHERE q.questionid = f.questionid)
HAVING count(*) > 0
UNION ALL
SELECT 'orphaned_functionoptions', count(*)
  FROM public.functionoptions fo
 WHERE NOT EXISTS (SELECT 1 FROM public.functions f WHERE f.functionid = fo.functionid)
HAVING count(*) > 0
UNION ALL
SELECT 'orphaned_score_revisions', count(*)
  FROM public.score_revisions r
 WHERE NOT EXISTS (SELECT 1 FROM public.functions f WHERE f.functionid = r.functionid)
HAVING count(*) > 0;

-- After 0068: the same checks its migration logs as QUESTIONNAIRE_VERSION_ANOMALY.
SELECT to_regprocedure('public.questionnaire_version_anomalies()') IS NOT NULL AS has_anomalies \gset
\if :has_anomalies
SELECT check_name, n FROM public.questionnaire_version_anomalies() WHERE n > 0;
\endif
