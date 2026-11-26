// Review decisions for operations whose product calls are not all direct client imports.
module.exports = {
  vue: {
    workflow: {
      getMe: 'Map the session response into the shared user-store model.',
      getEveSsoLogin: 'Build the hash callback URL and scope list for browser navigation.',
      getEveSsoBind: 'Build the hash callback URL and scope list for character binding.',
      createAdmission: 'Shared create/edit/resubmit flow serializes the multipart payload.',
      editAdmission: 'Shared create/edit/resubmit flow serializes the multipart payload.',
      resubmitAdmission: 'Shared create/edit/resubmit flow serializes the multipart payload.',
      getAdmissionEvidence: 'Distinguish the binary response from business errors and validate the image media type.'
    },
    navigation: {
      eveSsoCallback: 'EVE SSO provider redirects the browser to the contract route; do not fetch it from the SPA.'
    }
  },
  react: {
    workflow: {
      getMe: 'Map the session response into the shared authentication model.',
      getEveSsoLogin: 'Build the browser callback URL and scope list for login.',
      getEveSsoBind: 'Build the browser callback URL and scope list for character binding.',
      createAdmission: 'Shared create/edit/resubmit flow serializes the multipart payload.',
      editAdmission: 'Shared create/edit/resubmit flow serializes the multipart payload.',
      resubmitAdmission: 'Shared create/edit/resubmit flow serializes the multipart payload.',
      getAdmissionEvidence: 'Distinguish binary responses from business errors and validate the media type.'
    },
    navigation: {
      eveSsoCallback: 'EVE SSO provider redirects the browser to the contract route; do not fetch it from the SPA.'
    }
  }
}
