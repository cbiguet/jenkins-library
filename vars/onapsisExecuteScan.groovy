import com.cloudbees.groovy.cps.NonCPS
import com.sap.piper.Utils


@Field String STEP_NAME = getClass().getName()
@Field String METADATA_FILE = 'metadata/onapsisExecuteScan.yaml'

/**
 * Name of library step
 *
 * @param script global script environment of the Jenkinsfile run
 * @param others document all parameters
 */
def call(Map parameters = [:], body) {
    handlePipelineStepErrors (stepName: STEP_NAME, stepParameters: parameters) {
        List credentials = [[type: 'token', id: 'onapsisCredentialsId', , env: ['PIPER_token']]]
        piperExecuteBin(parameters, STEP_NAME, METADATA_FILE, credentials)
    }
}
