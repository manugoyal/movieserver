/*
Copyright 2013 Manu Goyal

Licensed under the Apache License, Version 2.0 (the "License"); you may not use
this file except in compliance with the License.  You may obtain a copy of the
License at

http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software distributed
under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR
CONDITIONS OF ANY KIND, either express or implied.  See the License for the
specific language governing permissions and limitations under the License.
 */

requirejs.config({
  baseUrl: 'static/js/',

  paths: {
    'jquery':                'libs/jquery',
    'underscore':            'libs/underscore',
    'backbone':              'libs/backbone/backbone',
    'backbone_pageable':     'libs/backbone/backbone-pageable',
    'text':                  'libs/text',
    'backgrid':              'libs/backgrid/backgrid',
    'backgrid_paginator':    'libs/backgrid/backgrid-paginator-custom',
    'backgrid_filter':       'libs/backgrid/backgrid-filter-custom',
    'lunr':                  'libs/lunr'
  },

  shim: {
    'jquery': { exports: '$' },
    'underscore': { exports: '_' },

    'backbone': {
      deps: ['jquery', 'underscore'],
      exports: 'Backbone'
    },

    'backbone_pageable': {
      'deps': ['backbone']
    },

    'backgrid': {
      deps: ['backbone', 'backbone_pageable'],
      exports: 'Backgrid'
    },

    'backgrid_paginator': {
      deps: ['backgrid']
    },

    'backgrid_filter': {
      deps: ['backgrid', 'lunr']
    }

  }
});

requirejs(['app'], function(App) {
  App.initialize();
});
